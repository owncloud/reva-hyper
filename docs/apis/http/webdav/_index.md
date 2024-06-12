---
title: "WebDAV"
date: 2023-07-02T00:00:00+00:00
weight: 20
geekdocRepo: https://github.com/owncloud/ocis
geekdocEditPath: edit/master/docs/apis/http/webdav
geekdocFilePath: _index.md
geekdocCollapseSection: true
---

{{< toc >}}

**Web** **D**istributed **A**uthoring and **V**ersioning (WebDAV) consists of a set of methods, headers, and content-types extending HTTP/1.1 for the management of resources and -properties, creation and management of resource collections, URL namespace manipulation, and resource locking (collision avoidance). WebDAV is one of the central APIs that ownCloud uses for handling file resources, metadata and locks.


{{< hint type=info title="RFC" >}}
**WebDAV RFCs**

RFC 2518 was published in February 1999. [RFC 4918](https://datatracker.ietf.org/doc/html/rfc4918), published in June 2008 obsoletes RFC 2518 with minor revisions mostly due to interoperability experience.

{{< /hint >}}
## Calling the WebDAV API

### Request URI

```sh
{HTTP method} https://ocis.url/{webdav-base}/{resourceID}/{path}
```

The request URI consists of:

| Component     | Description                                                                                            |
|---------------|--------------------------------------------------------------------------------------------------------|
| {HTTP method} | The HTTP method which is used in the request.                                                          |
| {webdav-base} | The WebDAV base path component. Possible options are                                                   |
|               | `dav/spaces/` This is the default and optimized endpoint for all WebDAV requests.                      |
|               | `remote.php/dav/spaces/`*                                                                              |
|               | `remote.php/webdav/`*                                                                                  |
|               | `webdav/`*                                                                                             |
|               | `dav/`*                                                                                                |
| {resourceID}  | This resourceID is used as the WebDAV root element. All children are accessed by their relative paths. |
| {path}        | The relative path to the WebDAV root. In most of the casese, this is the space root.                   |

\* these dav endpoints are implemented for legacy reasons and should not be used. Note: The legacy endpoints **do not take the resourceID as an argument.**

### HTTP methods

| Method    | Description                                                                                                                                                                                                                                  |
|-----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| PROPFIND  | Retrieve properties as XML from a web resource. It is also overloaded to retrieve the collection structure (a.k.a. directory hierarchy) of a remote system.                                                                                  |
| PROPPATCH | Process instructions specified in the request body to set and/or remove properties defined on the resource identified by the request uri.                                                                                                    |
| MKCOL     | Create a WebDAV collection (folder) at the location specified by the request uri.                                                                                                                                                            |
| GET       | Retrieve a WebDAV resource.                                                                                                                                                                                                                  |
| HEAD      | Retrieve a WebDAV resource without reading the body.                                                                                                                                                                                         |
| PUT       | A PUT performed on an existing resource replaces the GET response entity of the resource.                                                                                                                                                    |
| POST      | Not part of the WebDAV rfc and has no effect on a WebDAV resource. However, this method is used in the TUS protocol for uploading resources.                                                                                      |
| PATCH     | Not part of the WebDAV rfc and has no effect on a WebDAV resource. However, this method is used in the TUS protocol for uploading resources.                                                                                           |
| COPY      | Creates a duplicate of the source resource identified by the Request-URI, in the destination resource identified by the URI in the Destination header.                                                                                       |
| MOVE      | The MOVE operation on a non-collection resource is the logical equivalent of a copy (COPY), followed by consistency maintenance processing, followed by a delete of the source, where all three actions are performed in a single operation. |                                                                                                                             |
| DELETE    | Delete the resource identified by the Request-URI.                                                                                                                                                                                           |
| LOCK      | A LOCK request to an existing resource will create a lock on the resource identified by the Request-URI, provided the resource is not already locked with a conflicting lock.                                                                |
| UNLOCK    | The UNLOCK method removes the lock identified by the lock token in the Lock-Token request header. The Request-URI must identify a resource within the scope of the lock.                                                                     |

The methods `MKCOL`, `GET`, `HEAD`, `LOCK`, `COPY`, `MOVE`, `UNLOCK` and `DELETE` need no request body.

The methods `PROPFIND`, `PROPPATCH`, `PUT` require a request body, normally in XML format to provide the needed values.

{{< hint type=tip title="Tooling" >}}
**WebDAV is not REST**

The WebDAV protocol was created before the REST paradigm has become the de-facto standard for API design. WebDAV uses http methods which are not part of REST. Therefore all the tooling around API design and documentation is not usable (like OpenApi 3.0 / Swagger or others).
{{< /hint >}}

### Authentication

For development purposes the examples in the developer documentation use Basic Auth. It is disabled by default and should only be enabled by setting `PROXY_ENABLE_BASIC_AUTH` in [the proxy](../../../services/proxy/configuration/#environment-variables) for development or test instances.

To authenticate with a Bearer token or OpenID Connect access token replace the `-u user:password` Basic Auth option of curl with a `-H 'Authorization: Bearer <token>'` header. A `<token>` can be obtained by copying it from a request in the browser, although it will time out within minutes. To automatically refresh the OpenID Connect access token an ssh-agent like solution like [oidc-agent](https://github.com/indigo-dc/oidc-agent) should be used.

## Listing Properties

This method is used to list the properties of a resource in xml. This method can also be used to retrieve the listing of a WebDAV collection which means the content of a remote directory.

{{< tabs "list-properties" >}}
{{< tab "Curl" >}}
```shell
curl -L -X PROPFIND 'https://localhost:9200/dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/' \
-H 'Depth: 1' \
-d '<?xml version="1.0"?>
<d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
  <d:prop>
    <oc:permissions />
    <oc:favorite />
    <oc:fileid />
    <oc:owner-id />
    <oc:owner-display-name />
    <oc:share-types />
    <oc:privatelink />
    <d:getcontentlength />
    <oc:size />
    <d:getlastmodified />
    <d:getetag />
    <d:getcontenttype />
    <d:resourcetype />
    <oc:downloadURL />
  </d:prop>
</d:propfind>'
```
{{< /tab >}}
{{< tab "HTTP" >}}
```shell
PROPFIND /dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/ HTTP/1.1
Host: localhost:9200
Origin: https://localhost
Access-Control-Request-Method: PROPFIND
Depth: 1
Content-Type: application/xml
Authorization: Basic YWRtaW46YWRtaW4=
Content-Length: 436

<?xml version="1.0"?>
<d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
  <d:prop>
    <oc:permissions />
    <oc:favorite />
    <oc:fileid />
    <oc:owner-id />
    <oc:owner-display-name />
    <oc:share-types />
    <oc:privatelink />
    <d:getcontentlength />
    <oc:size />
    <d:getlastmodified />
    <d:getetag />
    <d:getcontenttype />
    <d:resourcetype />
    <oc:downloadURL />
  </d:prop>
</d:propfind>
```
{{< /tab >}}
{{< /tabs >}}

The request consists of a request body and an optional `Depth` Header.

{{< hint type=tip title="PROPFIND usage" >}}
**Metadata and Directory listings**

Clients can use the `PROPFIND` method to retrieve properties of resources (metadata) and to list the content of a directories.
{{< /hint >}}
### Response

{{< tabs "response list properties" >}}
{{< tab "207 - Multistatus" >}}

#### Multi Status Response

A Multi-Status response conveys information about multiple resources
in situations where multiple status codes might be appropriate.  The
default Multi-Status response body is an application/xml
HTTP entity with a `multistatus` root element.  Further elements
contain `200`, `300`, `400`, and `500` series status codes generated during
the method invocation.

Although `207` is used as the overall response status code, the
recipient needs to consult the contents of the multistatus response
body for further information about the success or failure of the
method execution.  The response MAY be used in success, partial
success and also in failure situations.

The `multistatus` root element holds zero or more `response` elements
in any order, each with information about an individual resource.

#### Body

```xml
<d:multistatus xmlns:s="http://sabredav.org/ns" xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
    <d:response>
        <d:href>/dav/spaces/storage-users-1$some-admin-user-id-0000-000000000000/</d:href>
        <d:propstat>
            <d:prop>
                <oc:permissions>RDNVCKZP</oc:permissions>
                <oc:favorite>0</oc:favorite>
                <oc:fileid>storage-users-1$some-admin-user-id-0000-000000000000!some-admin-user-id-0000-000000000000</oc:fileid>
                <oc:id>storage-users-1$some-admin-user-id-0000-000000000000!some-admin-user-id-0000-000000000000</oc:id>
                <oc:owner-id>admin</oc:owner-id>
                <oc:owner-display-name>Admin</oc:owner-display-name>
                <oc:privatelink>https://localhost:9200/f/storage-users-1$some-admin-user-id-0000-000000000000%21some-admin-user-id-0000-000000000000</oc:privatelink>
                <oc:size>10364682</oc:size>
                <d:getlastmodified>Mon, 04 Sep 2023 20:10:09 GMT</d:getlastmodified>
                <d:getetag>"c4d3610dfe4fac9b44e1175cfc44b12b"</d:getetag>
                <d:resourcetype>
                    <d:collection/>
                </d:resourcetype>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
        <d:propstat>
            <d:prop>
                <oc:checksums></oc:checksums>
                <oc:share-types></oc:share-types>
                <d:getcontentlength></d:getcontentlength>
                <d:getcontenttype></d:getcontenttype>
            </d:prop>
            <d:status>HTTP/1.1 404 Not Found</d:status>
        </d:propstat>
    </d:response>
    <d:response>
        <d:href>/dav/spaces/storage-users-1$some-admin-user-id-0000-000000000000/New%20file.txt</d:href>
        <d:propstat>
            <d:prop>
                <oc:permissions>RDNVWZP</oc:permissions>
                <oc:checksums>
                    <oc:checksum>SHA1:1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea MD5:2205e48de5f93c784733ffcca841d2b5 ADLER32:058801ab</oc:checksum>
                </oc:checksums>
                <oc:favorite>0</oc:favorite>
                <oc:fileid>storage-users-1$some-admin-user-id-0000-000000000000!90cc3e73-0c6c-4346-9c4d-f529976d4990</oc:fileid>
                <oc:id>storage-users-1$some-admin-user-id-0000-000000000000!90cc3e73-0c6c-4346-9c4d-f529976d4990</oc:id>
                <oc:owner-id>admin</oc:owner-id>
                <oc:owner-display-name>Admin</oc:owner-display-name>
                <oc:share-types>
                    <oc:share-type>0</oc:share-type>
                    <oc:share-type>1</oc:share-type>
                    <oc:share-type>3</oc:share-type>
                </oc:share-types>
                <oc:privatelink>https://localhost:9200/f/storage-users-1$some-admin-user-id-0000-000000000000%2190cc3e73-0c6c-4346-9c4d-f529976d4990</oc:privatelink>
                <d:getcontentlength>5</d:getcontentlength>
                <oc:size>5</oc:size>
                <d:getlastmodified>Mon, 28 Aug 2023 20:45:03 GMT</d:getlastmodified>
                <d:getetag>"75115347c74701a3be9c635ddebbf5c4"</d:getetag>
                <d:getcontenttype>text/plain</d:getcontenttype>
                <d:resourcetype></d:resourcetype>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
    </d:response>
    <d:response>
        <d:href>/dav/spaces/storage-users-1$some-admin-user-id-0000-000000000000/NewFolder/</d:href>
        <d:propstat>
            <d:prop>
                <oc:permissions>RDNVCKZP</oc:permissions>
                <oc:favorite>0</oc:favorite>
                <oc:fileid>storage-users-1$some-admin-user-id-0000-000000000000!5c73ecd9-d9f4-44f4-b685-ca4cb40aa6b7</oc:fileid>
                <oc:id>storage-users-1$some-admin-user-id-0000-000000000000!5c73ecd9-d9f4-44f4-b685-ca4cb40aa6b7</oc:id>
                <oc:owner-id>admin</oc:owner-id>
                <oc:owner-display-name>Admin</oc:owner-display-name>
                <oc:privatelink>https://localhost:9200/f/storage-users-1$some-admin-user-id-0000-000000000000%215c73ecd9-d9f4-44f4-b685-ca4cb40aa6b7</oc:privatelink>
                <oc:size>0</oc:size>
                <d:getlastmodified>Mon, 28 Aug 2023 20:45:10 GMT</d:getlastmodified>
                <d:getetag>"e83367534cc595a45d706857fa5f03d8"</d:getetag>
                <d:resourcetype>
                    <d:collection/>
                </d:resourcetype>
            </d:prop>
            <d:status>HTTP/1.1 200 OK</d:status>
        </d:propstat>
        <d:propstat>
            <d:prop>
                <oc:checksums></oc:checksums>
                <oc:share-types></oc:share-types>
                <d:getcontentlength></d:getcontentlength>
                <d:getcontenttype></d:getcontenttype>
            </d:prop>
            <d:status>HTTP/1.1 404 Not Found</d:status>
        </d:propstat>
    </d:response>
</d:multistatus>
```
{{< /tab >}}
{{< tab "400 - Bad Request" >}}

#### Body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<d:error xmlns:d="DAV" xmlns:s="http://sabredav.org/ns">
    <s:exception>Sabre\DAV\Exception\BadRequest</s:exception>
    <s:message>Invalid Depth header value: 3</s:message>
</d:error>
```

This can occur if the request is malformed e.g. due to an invalid xml request body or an invalid depth header value.
{{< /tab >}}
{{< tab "404 - Not Found" >}}

#### Body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<d:error xmlns:d="DAV" xmlns:s="http://sabredav.org/ns">
    <s:exception>Sabre\DAV\Exception\NotFound</s:exception>
    <s:message>Resource not found</s:message>
</d:error>
```
{{< /tab >}}
{{< /tabs >}}

### Request Body

The `PROPFIND` Request can include an XML request body containing a list of namespaced property names.

### Namespaces

When building the body of your DAV request, you will request properties that are available under a specific namespace URI. It is usual to declare prefixes for those namespace in the `d:propfind` element of the body.

Available namespaces:

| URI                                       | Prefix |
|-------------------------------------------|--------|
| DAV:                                      | d      |
| http://sabredav.org/ns                    | s      |
| http://owncloud.org/ns                    | oc     |
| http://open-collaboration-services.org/ns | ocs    |
| http://open-cloud-mesh.org/ns             | ocm    |

### Request Example with declared namespaces

```xml
<?xml version="1.0"?>
    <d:propfind xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
    </d:propfind>
```

### Supported WebDAV Properties

| Property                            | Desription                                                                 | Example                                                                                                                                          |
| ----------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| `<d:getlastmodified />`             | The latest modification time.                                              | `Fri, 30 Dec 2022 14:22:43 GMT`                                                                                                                  |
| `<d:getetag />`                     | The file's etag.                                                           | `"c3a1ee4a0c28edc15b9635c3bf798013"`                                                                                                             |
| `<d:getcontenttype />`              | The mime type of the file.                                                 | `image/jpeg`                                                                                                                                     |
| `<d:resourcetype />`                | Specifies the nature of the resource.                                      | `<d:collection />` for a folder                                                                                                                  |
| `<d:getcontentlength />`            | The size if it is a file in bytes.                                         | `5` bytes                                                                                                                                        |
| `<d:lockdiscovery />`               | Describes the active locks on a resource.                                  | Detailed Example in [Locking]()                                                                                                                  |
| `<oc:id />`                         | The globally unique ID of the resource.                                    | `storage-1$27475553-7fb7-4689-b4cf-bbb635daff79!27475553-7fb7-4689-b4cf-bbb635daff79`                                                            |
| `<oc:fileid />`                     | The globally unique ID of the resource.                                    | `storage-1$27475553-7fb7-4689-b4cf-bbb635daff79!27475553-7fb7-4689-b4cf-bbb635daff79`                                                            |
| `<oc:downloadURL />`                | Direct URL to download a file from.                                        | Not implemented.                                                                                                                                 |
| `<oc:permissions />`                | Determines the actions a user can take on the resource.                    | The value is a string containing letters that clients can use to determine available actions.                                                    |
|                                     |                                                                            | `S`: Shared                                                                                                                                      |
|                                     |                                                                            | `M`: Mounted                                                                                                                                     |
|                                     |                                                                            | `D`: Deletable                                                                                                                                   |
|                                     |                                                                            | `NV`: Updateable, Renameable, Moveable                                                                                                           |
|                                     |                                                                            | `W`: Updateable (file)                                                                                                                           |
|                                     |                                                                            | `CK`: Creatable (folders only)                                                                                                                   |
|                                     |                                                                            | `Z`: Deniable                                                                                                                                    |
|                                     |                                                                            | `P`: Trashbin Purgable                                                                                                                           |
|                                     |                                                                            | `X`: Securely Viewable                                                                                                                           |
|                                     |                                                                            | In the early stages this was indeed a list of permissions. Over time, more flags were added and the term permissions no longer really fits well. |
| `<oc:tags />`                       | List of user specified tags.                                               | `<oc:tag>test</oc:tag>`                                                                                                                          |
| `<oc:favorite />	`                  | The favorite state.                                                        | `0` for not favourited, `1` for favourited                                                                                                       |
| `<oc:owner-id />`                   | The user id of the owner of a resource. Project spaces have no owner.      | `einstein`                                                                                                                                       |
| `<oc:owner-display-name />`         | The display name of the owner of a resource. Project spaces have no owner. | `Albert Einstein`                                                                                                                                |
| `<oc:share-types />`                | List of share types.                                                       | `0` = User Share                                                                                                                                 |
|                                     |                                                                            | `1` = Group Share                                                                                                                                |
|                                     |                                                                            | `2` = Public Link                                                                                                                                |
| `<oc:checksums />`                  |                                                                            | `<oc:checksum>`<br/>`SHA1:1c68ea370b40c06fcaf7f26c8b1dba9d9caf5dea MD5:2205e48de5f93c784733ffcca841d2b5 ADLER32:058801ab`<br /> `</oc:checksum>` |
|                                     |                                                                            | Due to a bug in the very early development of ownCloud, this value is not an array, but a string separated by whitespaces.                       |
| `<oc:size />`                       | Similar to `getcontentlength` but it also works for folders.               | `10` bytes                                                                                                                                       |
| `<oc:shareid />`                    | The ID of the share if the resource is part of such.                       | `storage-1$27475553-7fb7-4689-b4cf-bbb635daff79!27475553-7fb7-4689-b4cf-bbb635daff79`                                                            |
| `<oc:shareroot />`                  | The root path of the shared resource if the resource is part of such.      | `/shared-folder`                                                                                                                                 |
| `<oc:remoteItemId />`               | The ID of the shared resource if the resource is part of such.             | `storage-1$27475553-7fb7-4689-b4cf-bbb635daff79!27475553-7fb7-4689-b4cf-bbb635daff79`                                                            |
| `<oc:public-link-item-type />`      | The type of the resource if it's a public link.                            | `folder`                                                                                                                                         |
| `<oc:public-link-permission />`     | The share permissions of the resource if it's a public link.               | `1`                                                                                                                                              |
| `<oc:public-link-type />`           | The libregraph public share LinkType representation.                       | `view, edit` for type file, `view, edit, upload, createOnly` for type folder                                                                     |
| `<oc:public-link-expiration />`     | The expiration date of the public link.                                    | `Tue, 14 May 2024 12:44:29 GMT`                                                                                                                  |
| `<oc:public-link-share-datetime />` | The date the public link was created.                                      | `Tue, 14 May 2024 12:44:29 GMT`                                                                                                                  |
| `<oc:public-link-share-owner />`    | The username of the user who created the public link.                      | `admin`                                                                                                                                          |
| `<oc:trashbin-original-filename />` | The original name of the resource before it was deleted.                   | `some-file.txt`                                                                                                                                  |
| `<oc:trashbin-original-location />` | The original location of the resource before it was deleted.               | `some-file.txt`                                                                                                                                  |
| `<oc:trashbin-delete-datetime />`   | The date the resource was deleted.                                         | `Tue, 14 May 2024 12:44:29 GMT`                                                                                                                  |
| `<oc:audio />`                      | Audio meta data if the resource contains such.                             | `<oc:artist>Metallica</oc:artist><oc:album>Metallica</oc:album><oc:title>Enter Sandman</oc:title>`                                               |
| `<oc:location />`                   | Location meta data if the resource contains such.                          | `<oc:latitude>51.504106</oc:latitude><oc:longitude>-0.074575</oc:latitude>`                                                                      |

### Request Headers

A client executing a `PROPFIND` request MUST submit a Depth Header value. In practice, support for infinite-depth requests MAY be disabled, due to the performance and security concerns associated with this behavior.  Servers SHOULD treat a
request without a Depth header as if a `Depth: infinity` header was included. Infinite depth requests are disabled by default in ocis.

| Name                                      | Value                                                                                 |
|-------------------------------------------|---------------------------------------------------------------------------------------|
| Depth                                     | `0` = Only return the desired resource.                                               |
|                                           | `1` = Return the desired resource and all resources one level below in the hierarchy. |
|                                           | `infinity` = Return all resources below the root.                                     |

{{< hint type=caution title="Use the Depth header with caution" >}}
**Depth: infinity**

Using the `Depth: infinity` header value can cause heavy load on the server, depending on the size of the file tree.

The request can run into a timeout and the server performance could be affected for other users.

{{< /hint >}}

## Create a Directory

Clients create directories (WebDAV collections) by executing a `MKCOL` request at the location specified by the request url.

{{< tabs "create-folder" >}}
{{< tab "Curl" >}}
```shell
curl -L -X MKCOL 'https://localhost:9200/dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/NewFolder/' \
-H 'Authorization: Basic YWRtaW46YWRtaW4='
```
{{< /tab >}}
{{< tab "HTTP" >}}
```shell
MKCOL /dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/NewFolder/ HTTP/1.1
Host: localhost:9200
Authorization: Basic YWRtaW46YWRtaW4=
```
{{< /tab >}}
{{< /tabs >}}
### Response

{{< tabs "response create folder" >}}
{{< tab "201 - Created" >}}
This indicates that the Resource has been created successfully.

#### Body

The response has no body.
{{< /tab >}}
{{< tab "403 - Forbidden" >}}

#### Body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<d:error xmlns:d="DAV" xmlns:s="http://sabredav.org/ns">
    <s:exception>Sabre\DAV\Exception\Forbidden</s:exception>
    <s:message></s:message>
</d:error>
```
{{< /tab >}}
{{< tab "405 - Method not allowed" >}}

#### Body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<d:error xmlns:d="DAV" xmlns:s="http://sabredav.org/ns">
    <s:exception>Sabre\DAV\Exception\MethodNotAllowed</s:exception>
    <s:message>The resource you tried to create already exists</s:message>
</d:error>
```
{{< /tab >}}
{{< /tabs >}}

## Upload File

To upload files to the remote server, clients can use the `PUT` method to create or fully replace the content of the remote file.

### Request Headers

| Name          | Usage                                                                                                                                                                                                                                                                                                               |
|---------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `X-OC-Mtime`  | Send the last modified <br> time of the file to the server in unixtime format. The server applies this mtime to the resource rather than the actual time.                                                                                                                                                           |
| `OC-Checksum` | Provide the checksum of the <br> file content to the server. <br> This is used to prevent corrupted data transfers.                                                                                                                                                                                                 |
| `If-Match`    | The If-Match request-header field is used with a method to make it <br> conditional. A client that has one or more entities previously <br> obtained from the resource can verify that one of those entities is <br> current by including a list of their associated entity tags in the <br> If-Match header field. |

{{< tabs "upload-file" >}}
{{< tab "Curl" >}}
```shell
curl -L -X PUT 'https://localhost:9200/dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/test.txt' \
-H 'X-OC-Mtime: 1692369418' \
-H 'OC-Checksum: SHA1:40bd001563085fc35165329ea1ff5c5ecbdbbeef' \
-H 'If-Match: "4436aef907f41f1ac7dfd1ac3d0d455f"' \
-H 'Content-Type: text/plain' \
-H 'Authorization: Basic YWRtaW46YWRtaW4=' \
-d '123'
```
{{< /tab >}}
{{< tab "HTTP" >}}
```shell
PUT /dav/spaces/storage-users-1%24some-admin-user-id-0000-000000000000/test.txt HTTP/1.1
Host: localhost:9200
X-OC-Mtime: 1692369418
OC-Checksum: SHA1:40bd001563085fc35165329ea1ff5c5ecbdbbeef
If-Match: "4436aef907f41f1ac7dfd1ac3d0d455f"
Content-Type: text/plain
Authorization: Basic YWRtaW46YWRtaW4=
Content-Length: 3

123
```
{{< /tab >}}
{{< /tabs >}}

### Response

{{< tabs "response upload file" >}}
{{< tab "201 - Created" >}}
This indicates that the Resource has been created successfully.

#### Body

The response has no body.

#### Headers

```yaml
Oc-Etag: "4436aef907f41f1ac7dfd1ac3d0d455f"
Oc-Fileid: storage-users-1$some-admin-user-id-0000-000000000000!07452b22-0ba9-4539-96e1-3511aff7fd2f
Last-Modified: Fri, 18 Aug 2023 14:36:58 +0000
X-Oc-Mtime: accepted
```
{{< /tab >}}
{{< tab "204 - No Content" >}}
This indicates that the Resource has been updated successfully.

#### Body

The response has no body.

#### Headers

```yaml
Oc-Etag: "4436aef907f41f1ac7dfd1ac3d0d455f"
Oc-Fileid: storage-users-1$some-admin-user-id-0000-000000000000!07452b22-0ba9-4539-96e1-3511aff7fd2f
Last-Modified: Fri, 18 Aug 2023 14:36:58 +0000
X-Oc-Mtime: accepted
```
{{< /tab >}}
{{< tab "400 - Bad Request" >}}
This indicates that the checksum, which was sent by the client, does not match the computed one after all bytes have been received by the server.

#### Body

```xml
<?xml version="1.0" encoding="UTF-8"?>
<d:error xmlns:d="DAV" xmlns:s="http://sabredav.org/ns">
    <s:exception>Sabre\DAV\Exception\BadRequest</s:exception>
    <s:message>The computed checksum does not match the one received from the client.</s:message>
</d:error>
```
{{< /tab >}}
{{< tab "403 - Forbidden" >}}

The user cannot create files in that remote location.
{{< /tab >}}
{{< tab "404 - Not Found" >}}

The remote target space cannot be found.
{{< /tab >}}
{{< tab "409 - Conflict" >}}

This error can occur when the request cannot be executed due to a missing precondition. One example is a PUT into a non-existing remote folder. It can also happen when the client sends the wrong etag in the `If-Match` header.
{{< /tab >}}
{{< /tabs >}}
