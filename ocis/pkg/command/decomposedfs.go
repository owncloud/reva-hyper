package command

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	provider "github.com/cs3org/go-cs3apis/cs3/storage/provider/v1beta1"
	revactx "github.com/cs3org/reva/v2/pkg/ctx"
	"github.com/cs3org/reva/v2/pkg/storage/cache"
	"github.com/cs3org/reva/v2/pkg/storage/fs/ocis/blobstore"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/lookup"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/metadata"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/metadata/prefixes"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/node"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/options"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/spaceidindex"
	"github.com/cs3org/reva/v2/pkg/storage/utils/decomposedfs/tree"
	"github.com/cs3org/reva/v2/pkg/storagespace"
	"github.com/cs3org/reva/v2/pkg/store"
	"github.com/owncloud/ocis/v2/ocis-pkg/config"
	"github.com/owncloud/ocis/v2/ocis/pkg/register"
	"github.com/urfave/cli/v2"
)

// DecomposedfsCommand is the entrypoint for the groups command.
func DecomposedfsCommand(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:     "decomposedfs",
		Usage:    `cli tools to inspect and manipulate a decomposedfs storage.`,
		Category: "maintenance",
		Subcommands: []*cli.Command{
			metadataCmd(cfg),
			checkCmd(cfg),
			indexCmd(cfg),
		},
	}
}

func init() {
	register.AddCommand(DecomposedfsCommand)
}

func checkCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "check-treesize",
		Usage: `cli tool to check the treesize metadata of a Space`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "root",
				Aliases:  []string{"r"},
				Required: true,
				Usage:    "Path to the root directory of the decomposedfs",
			},
			&cli.StringFlag{
				Name:     "node",
				Required: true,
				Aliases:  []string{"n"},
				Usage:    "Space ID of the Space to inspect",
			},
			&cli.BoolFlag{
				Name:  "repair",
				Usage: "Try to repair nodes with incorrect treesize metadata. IMPORTANT: Only use this while ownCloud Infinite Scale is not running.",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "Do not prompt for confirmation when running in repair mode.",
			},
		},
		Action: check,
	}
}

func check(c *cli.Context) error {
	rootFlag := c.String("root")
	repairFlag := c.Bool("repair")

	if repairFlag && !c.Bool("force") {
		answer := strings.ToLower(stringPrompt("IMPORTANT: Only use '--repair' when ownCloud Infinite Scale is not running. Do you want to continue? [yes | no = default]"))
		if answer != "yes" && answer != "y" {
			return nil
		}
	}

	lu, backend := getBackend(c)
	o := &options.Options{
		MetadataBackend: backend.Name(),
		MaxConcurrency:  100,
	}
	bs, err := blobstore.New(rootFlag)
	if err != nil {
		fmt.Println("Failed to init blobstore")
		return err
	}

	tree := tree.New(lu, bs, o, store.Create())

	nId := c.String("node")
	n, err := lu.NodeFromSpaceID(context.Background(), nId)
	if err != nil || !n.Exists {
		fmt.Println("Can not find node '" + nId + "'")
		return err
	}
	fmt.Printf("Checking treesizes in space: %s (id: %s)\n", n.Name, n.ID)
	ctx := revactx.ContextSetUser(context.Background(),
		&userpb.User{
			Id: &userpb.UserId{
				OpaqueId: "00000000-0000-0000-0000-000000000000",
			},
			Username: "offline",
		})

	treeSize, err := walkTree(ctx, tree, lu, n, repairFlag)
	treesizeFromMetadata, err := n.GetTreeSize(c.Context)
	if err != nil {
		fmt.Printf("failed to read treesize of node: %s: %s\n", n.ID, err)
	}
	if treesizeFromMetadata != treeSize {
		fmt.Printf("Tree sizes mismatch for space: %s\n\tNodeId: %s\n\tInternalPath: %s\n\tcalculated treesize: %d\n\ttreesize in metadata: %d\n",
			n.Name, n.ID, n.InternalPath(), treeSize, treesizeFromMetadata)
		if repairFlag {
			fmt.Printf("Fixing tree size for node: %s. Calculated treesize: %d\n",
				n.ID, treeSize)
			n.SetTreeSize(c.Context, treeSize)
		}
	}
	return nil
}

func walkTree(ctx context.Context, tree *tree.Tree, lu *lookup.Lookup, root *node.Node, repair bool) (uint64, error) {
	if root.Type(ctx) != provider.ResourceType_RESOURCE_TYPE_CONTAINER {
		return 0, errors.New("can't travers non-container nodes")
	}
	children, err := tree.ListFolder(ctx, root)
	if err != nil {
		fmt.Println("Can not list children for space'" + root.ID + "'")
		return 0, err
	}

	var treesize uint64
	for _, child := range children {
		switch child.Type(ctx) {
		case provider.ResourceType_RESOURCE_TYPE_CONTAINER:
			subtreesize, err := walkTree(ctx, tree, lu, child, repair)
			if err != nil {
				fmt.Printf("error calculating tree size of node: %s: %s\n", child.ID, err)
				return 0, err
			}
			treesizeFromMetadata, err := child.GetTreeSize(ctx)
			if err != nil {
				fmt.Printf("failed to read tree size of node: %s: %s\n", child.ID, err)
				return 0, err
			}
			if treesizeFromMetadata != subtreesize {
				origin, err := lu.Path(ctx, child, node.NoCheck)
				if err != nil {
					fmt.Printf("error get path: %s\n", err)
				}
				fmt.Printf("Tree sizes mismatch for node: %s\n\tNodeId: %s\n\tInternalPath: %s\n\tcalculated treesize: %d\n\ttreesize in metadata: %d\n",
					origin, child.ID, child.InternalPath(), subtreesize, treesizeFromMetadata)
				if repair {
					fmt.Printf("Fixing tree size for node: %s. Calculated treesize: %d\n",
						child.ID, subtreesize)
					child.SetTreeSize(ctx, subtreesize)
				}
			}
			treesize += subtreesize
		case provider.ResourceType_RESOURCE_TYPE_FILE:
			blobsize, err := child.GetBlobSize(ctx)
			if err != nil {
				fmt.Printf("error reading blobsize of node: %s: %s\n", child.ID, err)
				return 0, err
			}
			treesize += blobsize
		default:
			fmt.Printf("Ignoring type: %v, node: %s %s\n", child.Type(ctx), child.Name, child.ID)
		}
	}

	return treesize, nil
}

func metadataCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "metadata",
		Usage: `cli tools to inspect and manipulate node metadata`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "root",
				Aliases:  []string{"r"},
				Required: true,
				Usage:    "Path to the decomposedfs",
			},
			&cli.StringFlag{
				Name:     "node",
				Required: true,
				Aliases:  []string{"n"},
				Usage:    "Path to or ID of the node to inspect",
			},
		},
		Subcommands: []*cli.Command{dumpCmd(cfg), getCmd(cfg), setCmd(cfg)},
	}
}

func dumpCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "dump",
		Usage: `print the metadata of the given node. String attributes will be enclosed in quotes. Binary attributes will be returned encoded as base64 with their value being prefixed with '0s'.`,
		Action: func(c *cli.Context) error {
			lu, backend := getBackend(c)
			path, err := getPath(c, lu)
			if err != nil {
				return err
			}

			attribs, err := backend.All(c.Context, path)
			if err != nil {
				fmt.Println("Error reading attributes")
				return err
			}
			printAttribs(attribs, c.String("attribute"))
			return nil
		},
	}
}

func getCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: `print a specific attribute of the given node. String attributes will be enclosed in quotes. Binary attributes will be returned encoded as base64 with their value being prefixed with '0s'.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "attribute",
				Aliases: []string{"a"},
				Usage:   "attribute to inspect",
			},
		},
		Action: func(c *cli.Context) error {
			lu, backend := getBackend(c)
			path, err := getPath(c, lu)
			if err != nil {
				return err
			}

			attribs, err := backend.All(c.Context, path)
			if err != nil {
				fmt.Println("Error reading attributes")
				return err
			}
			printAttribs(attribs, c.String("attribute"))
			return nil
		},
	}
}

func setCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "set",
		Usage: `manipulate metadata of the given node. Binary attributes can be given hex encoded (prefix by '0x') or base64 encoded (prefix by '0s').`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "attribute",
				Required: true,
				Aliases:  []string{"a"},
				Usage:    "attribute to inspect",
			},
			&cli.StringFlag{
				Name:     "value",
				Required: true,
				Aliases:  []string{"v"},
				Usage:    "value to set",
			},
		},
		Action: func(c *cli.Context) error {
			lu, backend := getBackend(c)
			path, err := getPath(c, lu)
			if err != nil {
				return err
			}

			v := c.String("value")
			if strings.HasPrefix(v, "0s") {
				b64, err := base64.StdEncoding.DecodeString(v[2:])
				if err == nil {
					v = string(b64)
				} else {
					fmt.Printf("Error decoding base64 string: '%s'. Using as raw string.\n", err)
				}
			} else if strings.HasPrefix(v, "0x") {
				h, err := hex.DecodeString(v[2:])
				if err == nil {
					v = string(h)
				} else {
					fmt.Printf("Error decoding base64 string: '%s'. Using as raw string.\n", err)
				}
			}

			err = backend.Set(c.Context, path, c.String("attribute"), []byte(v))
			if err != nil {
				fmt.Println("Error setting attribute")
				return err
			}
			return nil
		},
	}
}

func backend(root, backend string) metadata.Backend {
	switch backend {
	case "xattrs":
		return metadata.XattrsBackend{}
	case "mpk":
		return metadata.NewMessagePackBackend(root, cache.Config{})
	}
	return metadata.NullBackend{}
}

func getBackend(c *cli.Context) (*lookup.Lookup, metadata.Backend) {
	rootFlag := c.String("root")

	bod := lookup.DetectBackendOnDisk(rootFlag)
	backend := backend(rootFlag, bod)
	lu := lookup.New(backend, &options.Options{
		Root:            rootFlag,
		MetadataBackend: bod,
	})
	return lu, backend
}

func getPath(c *cli.Context, lu *lookup.Lookup) (string, error) {
	nodeFlag := c.String("node")

	path := ""
	if strings.HasPrefix(nodeFlag, "/") {
		path = nodeFlag
	} else {
		nId := c.String("node")
		id, err := storagespace.ParseID(nId)
		if err != nil {
			fmt.Println("Invalid node id.")
			return "", err
		}
		n, err := lu.NodeFromID(context.Background(), &id)
		if err != nil || !n.Exists {
			fmt.Println("Can not find node '" + nId + "'")
			return "", err
		}
		path = n.InternalPath()
	}
	return path, nil
}

func printAttribs(attribs map[string][]byte, onlyAttribute string) {
	if onlyAttribute != "" {
		fmt.Println(onlyAttribute + `=` + attribToString(attribs[onlyAttribute]))
		return
	}

	names := []string{}
	for k := range attribs {
		names = append(names, k)
	}

	sort.Strings(names)

	for _, n := range names {
		fmt.Println(n + `=` + attribToString(attribs[n]))
	}
}

func attribToString(attrib []byte) string {
	for i := 0; i < len(attrib); i++ {
		if attrib[i] < 32 || attrib[i] >= 127 {
			return "0s" + base64.StdEncoding.EncodeToString(attrib)
		}
	}
	return `"` + string(attrib) + `"`
}

func indexCmd(cfg *config.Config) *cli.Command {
	return &cli.Command{
		Name:  "index",
		Usage: `cli tool to check the space indexes`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "root",
				Aliases:  []string{"r"},
				Required: true,
				Usage:    "Path to the root directory of the decomposedfs",
			},
			// TODO String flag for the type?
			// list values by default?
			&cli.BoolFlag{
				Name:  "repair",
				Usage: "add all spaces to the indexes",
			},
		},
		Action: index,
	}
}

func index(c *cli.Context) error {
	rootFlag := c.String("root")
	//repairFlag := c.Bool("repair")

	files, err := filepath.Glob(filepath.Join(rootFlag, "spaces", "*", "*"))
	if err != nil {
		return err
	}

	lu, _ := getBackend(c)

	byuserid := map[string]map[string]string{}
	bygroupid := map[string]map[string]string{}
	bytype := map[string]map[string]string{}

	for _, file := range files {
		spaceID := filepath.Base(filepath.Dir(file)) + filepath.Base(file)
		fmt.Println("space " + spaceID)
		n, err := node.ReadNode(c.Context, lu, spaceID, spaceID, true, nil, true)
		if err != nil {
			fmt.Println("  could not read node: " + err.Error())
			continue
		}
		spacetype, err := n.SpaceRoot.XattrString(c.Context, prefixes.SpaceTypeAttr)
		if err != nil {
			fmt.Println("  could not read space type: " + err.Error())
			continue
		}
		fmt.Println("  type " + spacetype)
		target := "../../../spaces/" + lookup.Pathify(spaceID, 1, 2) + "/nodes/" + lookup.Pathify(spaceID, 4, 2)
		if bytype[spacetype] == nil {
			bytype[spacetype] = make(map[string]string)
		}
		bytype[spacetype][spaceID] = target
		grants, err := n.ListGrants(c.Context)
		if err != nil {
			fmt.Println("  could not list grants: " + err.Error())
			continue
		}
		for _, grant := range grants {
			if grant.GetGrantee().GetUserId() != nil {
				userid := grant.GetGrantee().GetUserId().GetOpaqueId()
				fmt.Println("  user grant: " + grant.GetGrantee().GetUserId().GetOpaqueId())
				if byuserid[userid] == nil {
					byuserid[userid] = make(map[string]string)
				}
				byuserid[userid][spaceID] = target
			}
			if grant.GetGrantee().GetGroupId() != nil {
				groupid := grant.GetGrantee().GetGroupId().GetOpaqueId()
				fmt.Println("  group grant: " + groupid)
				if bygroupid[groupid] == nil {
					bygroupid[groupid] = make(map[string]string)
				}
				bygroupid[groupid][spaceID] = target
			}
		}
	}
	// TODO use a flag and lock to completely lock the indexes and rewrite them

	fmt.Println("rebuilding by-user-id indexes")
	userSpaceIndex := spaceidindex.New(filepath.Join(rootFlag, "indexes"), "by-user-id")
	err = userSpaceIndex.Init()
	if err != nil {
		return err
	}
	for userid, spaces := range byuserid {
		fmt.Print("  ", userid, " ", spaces, "\n")
		userSpaceIndex.AddAll(userid, spaces)
	}

	fmt.Println("rebuilding by-group-id indexes")
	groupSpaceIndex := spaceidindex.New(filepath.Join(rootFlag, "indexes"), "by-group-id")
	err = groupSpaceIndex.Init()
	if err != nil {
		return err
	}
	for groupid, spaces := range bygroupid {
		fmt.Print("  ", groupid, " ", spaces, "\n")
		groupSpaceIndex.AddAll(groupid, spaces)
	}

	fmt.Println("rebuilding by-type indexes")
	spaceTypeIndex := spaceidindex.New(filepath.Join(rootFlag, "indexes"), "by-type")
	err = spaceTypeIndex.Init()
	if err != nil {
		return err
	}
	for spaceType, spaces := range bytype {
		fmt.Print("  ", spaceType, " ", spaces, "\n")
		spaceTypeIndex.AddAll(spaceType, spaces)
	}
	// FIXME type might be share which requires iterating over all files or manually adding a space after dumping a list of shares from the share manager

	return nil
}
