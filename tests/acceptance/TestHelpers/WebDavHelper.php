<?php declare(strict_types=1);
/**
 * ownCloud
 *
 * @author Artur Neumann <artur@jankaritech.com>
 * @copyright Copyright (c) 2017 Artur Neumann artur@jankaritech.com
 *
 * This code is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License,
 * as published by the Free Software Foundation;
 * either version 3 of the License, or any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>
 *
 */
namespace TestHelpers;

use Exception;
use GuzzleHttp\Client;
use GuzzleHttp\Exception\GuzzleException;
use InvalidArgumentException;
use PHPUnit\Framework\Assert;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\StreamInterface;
use Symfony\Component\HttpFoundation\Response as HttpResponse;
use DateTime;

/**
 * Helper to make WebDav Requests
 *
 * @author Artur Neumann <artur@jankaritech.com>
 *
 */
class WebDavHelper {
	public const DAV_VERSION_OLD = 1;
	public const DAV_VERSION_NEW = 2;
	public const DAV_VERSION_SPACES = 3;

	/**
	 * @var array of users with their different space ids
	 */
	public static array $spacesIdRef = [];

	/**
	 * @return bool
	 */
	public static function withRemotePhp(): bool {
		// use remote.php by default
		return \getenv("WITH_REMOTE_PHP") !== "false";
	}

	/**
	 * @return bool
	 */
	public static function asyncPropagation(): bool {
		return \getenv("ASYNC_PROPAGATION") === "true";
	}

	/**
	 * async propagation delay in milliseconds
	 *
	 * @return int
	 */
	public static function asyncPropagationDelay(): int {
		$delay = (int) \getenv("ASYNC_PROPAGATION_DELAY_MS");
		if ($delay === 0) {
			// default delay: 100ms
			$delay = 100;
		}
		return $delay;
	}

	/**
	 * @param string $urlPath
	 *
	 * @return string
	 */
	public static function prefixRemotePhp(string $urlPath): string {
		if (self::withRemotePhp()) {
			return "remote.php/$urlPath";
		}
		return $urlPath;
	}

	/**
	 * @param string $url
	 *
	 * @return bool
	 */
	public static function isDAVRequest(string $url): bool {
		$found = \preg_match("/(\bwebdav\b|\bdav\b)/", $url);
		return (bool)$found;
	}

	/**
	 * @param int $extraWait
	 *
	 * @return void
	 */
	public static function waitForAsyncPropagation(int $extraWait): void {
		$delay = self::asyncPropagationDelay() + $extraWait;
		usleep($delay * 1000);
	}

	/**
	 * wait for async propagation after a request
	 * if the request is a DAV request and is successful
	 *
	 * @param string $url
	 * @param string $method
	 * @param int $statusCode
	 *
	 * @return void
	 */
	public static function waitAsyncPropagationAfterRequest(string $url, string $method, int $statusCode): void {
		$longerWaitMethods = ["MKCOL", "MOVE", "COPY", "DELETE"];
		$methods = \array_merge($longerWaitMethods, ["POST", "PUT"]);

		if ((WebdavHelper::isDAVRequest($url) || GraphHelper::isShareRequest($url))
			&& \str_starts_with($url, OcisHelper::getServerUrl())
			&& \in_array($method, $methods)
			&& $statusCode < 300
		) {
			$extraWait = 0;
			if (\in_array($method, $longerWaitMethods)) {
				// folder related requests require more time
				// wait longer for these methods
				// increase wait by 300ms
				$extraWait = 300;
			}
			self::waitForAsyncPropagation($extraWait);
		}
	}

	/**
	 * clear space id reference for user
	 *
	 * @param string|null $user
	 *
	 * @return void
	 * @throws Exception
	 */
	public static function removeSpaceIdReferenceForUser(
		?string $user
	):void {
		if (\array_key_exists($user, self::$spacesIdRef)) {
			unset(self::$spacesIdRef[$user]);
		}
	}

	/**
	 * @param string $namespaceString
	 *
	 * @return object
	 */
	public static function parseNamespace(string $namespaceString): object {
		//calculate the namespace prefix and namespace
		$matches = [];
		\preg_match("/^(.*)='(.*)'$/", $namespaceString, $matches);
		return (object)["namespace" => $matches[2], "prefix" => $matches[1]];
	}

	/**
	 * returns the id of a file
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersionToUse
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getFileIdForPath(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $spaceId = null,
		?string $xRequestId = '',
		?int $davPathVersionToUse = self::DAV_VERSION_NEW
	): string {
		$body
			= '<?xml version="1.0"?>
				<d:propfind  xmlns:d="DAV:" xmlns:oc="http://owncloud.org/ns">
 					<d:prop>
    					<oc:fileid />
  					</d:prop>
				</d:propfind>';
		$response = self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPFIND",
			$path,
			null,
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse
		);
		\preg_match(
			'/\<oc:fileid\>([^\<]*)\<\/oc:fileid\>/',
			$response->getBody()->getContents(),
			$matches
		);

		if (!isset($matches[1])) {
			throw new Exception("could not find fileId of $path");
		}

		return $matches[1];
	}

	/**
	 * returns body for propfind
	 *
	 * @param array|null $properties
	 *
	 * @return string
	 * @throws Exception
	 */
	public static function getBodyForPropfind(?array $properties): string {
		$propertyBody = "";
		$extraNamespaces = "";
		foreach ($properties as $namespaceString => $property) {
			if (\is_int($namespaceString)) {
				//default namespace prefix if the property has no array key
				//also used if no prefix is given in the property value
				$namespacePrefix = null;
			} else {
				$ns = self::parseNamespace($namespaceString);
				$namespacePrefix = $ns->prefix;
				$extraNamespaces .= " xmlns:$namespacePrefix=\"$ns->namespace\" ";
			}
			//if a namespace prefix is given in the property value use that
			if (\strpos($property, ":") !== false) {
				$propertyParts = \explode(":", $property);
				$namespacePrefix = $propertyParts[0];
				$property = $propertyParts[1];
			}

			if ($namespacePrefix) {
				$propertyBody .= "<$namespacePrefix:$property/>";
			} else {
				$propertyBody .= "<$property/>";
			}
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propfind
				   xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\"
				   xmlns:ocs=\"http://open-collaboration-services.org/ns\"
				   $extraNamespaces>
				    <d:prop>$propertyBody</d:prop>
				</d:propfind>";
		return $body;
	}

	/**
	 * sends a PROPFIND request
	 * with these registered namespaces:
	 *  | prefix | namespace                                 |
	 *  | d      | DAV:                                      |
	 *  | oc     | http://owncloud.org/ns                    |
	 *  | ocs    | http://open-collaboration-services.org/ns |
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string[] $properties
	 *        string can contain namespace prefix,
	 *        if no prefix is given 'd:' is used as prefix
	 *        if an associative array is used, then the key will be used as namespace
	 * @param string|null $xRequestId
	 * @param string|null $folderDepth
	 * @param string|null $spaceId
	 * @param string|null $type
	 * @param int|null $davPathVersionToUse
	 * @param string|null $doDavRequestAsUser
	 * @param array|null $headers
	 *
	 * @return ResponseInterface
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public static function propfind(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?array $properties,
		?string $xRequestId = '',
		?string $folderDepth = '1',
		?string $spaceId = null,
		?string $type = "files",
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $doDavRequestAsUser = null,
		?array $headers = []
	):ResponseInterface {
		$body = self::getBodyForPropfind($properties);
		$folderDepth = (string) $folderDepth;
		if ($folderDepth !== '0' && $folderDepth !== '1' && $folderDepth !== 'infinity') {
			if ($folderDepth !== '') {
				throw new InvalidArgumentException('Invalid depth value ' . $folderDepth);
			}
			$folderDepth = '1'; // oCIS server's default value
		}
		$headers['Depth'] = $folderDepth;
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPFIND",
			$path,
			$headers,
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse,
			$type,
			null,
			null,
			false,
			null,
			null,
			[],
			$doDavRequestAsUser
		);
	}

	/**
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $propertyName
	 * @param string|null $propertyValue
	 * @param string|null $xRequestId
	 * @param string|null $namespaceString string containing prefix and namespace
	 *                                     e.g "x1='http://whatever.org/ns'"
	 * @param int|null $davPathVersionToUse
	 * @param string|null $type
	 * @param string|null $spaceId
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function proppatch(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $propertyName,
		?string $propertyValue,
		?string $xRequestId = '',
		?string $namespaceString = null,
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $type="files",
		?string $spaceId = null,
	):ResponseInterface {
		if ($namespaceString !== null) {
			$ns = self::parseNamespace($namespaceString);
			$propertyBody = "<$ns->prefix:$propertyName" .
				" xmlns:$ns->prefix=\"$ns->namespace\">" .
				"$propertyValue" .
				"</$ns->prefix:$propertyName>";
		} else {
			$propertyBody = "<$propertyName>$propertyValue</$propertyName>";
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propertyupdate xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\">
				 <d:set>
				  <d:prop>$propertyBody</d:prop>
				 </d:set>
				</d:propertyupdate>";
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPPATCH",
			$path,
			[],
			$spaceId,
			$xRequestId,
			$body,
			$davPathVersionToUse,
			$type
		);
	}

	/**
	 * gets namespace-prefix, namespace url and propName from provided namespaceString or property
	 * or otherwise use default
	 *
	 * @param string $namespaceString
	 * @param string $property
	 *
	 * @return array
	 */
	public static function getPropertyWithNamespaceInfo(string $namespaceString = "", string $property = ""):array {
		$namespace = "";
		$namespacePrefix = "";
		if (\is_int($namespaceString)) {
			//default namespace prefix if the property has no array key
			//also used if no prefix is given in the property value
			$namespacePrefix = "d";
			$namespace = "DAV:";
		} elseif ($namespaceString) {
			$ns = self::parseNamespace($namespaceString);
			$namespacePrefix = $ns->prefix;
			$namespace = $ns->namespace;
		}
		//if a namespace prefix is given in the property value use that
		if ($property && \strpos($property, ":")) {
			$propertyParts = \explode(":", $property);
			$namespacePrefix = $propertyParts[0];
			$property = $propertyParts[1];
		}
		return [$namespacePrefix, $namespace, $property];
	}

	/**
	 * sends HTTP request PROPPATCH method with multiple properties
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string $path
	 * @param array|null $propertiesArray
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersion
	 * @param string|null $namespaceString
	 * @param string|null $type
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function proppatchWithMultipleProps(
		?string  $baseUrl,
		?string  $user,
		?string $password,
		string  $path,
		?array $propertiesArray,
		?string $xRequestId = '',
		?int  $davPathVersion = null,
		?string  $namespaceString = null,
		?string  $type="files"
	):ResponseInterface {
		$propertyBody = "";
		foreach ($propertiesArray as $propertyArray) {
			$property = $propertyArray["propertyName"];
			$value = $propertyArray["propertyValue"];

			if ($namespaceString !== null) {
				$matches = [];
				[$namespacePrefix, $namespace, $property] = self::getPropertyWithNamespaceInfo(
					$namespaceString,
					$property
				);
				$propertyBody .= "\n\t<$namespacePrefix:$property>" .
					"$value" .
					"</$namespacePrefix:$property>";
			} else {
				$propertyBody .= "<$property>$value</$property>";
			}
		}
		$body = "<?xml version=\"1.0\"?>
				<d:propertyupdate xmlns:d=\"DAV:\"
				   xmlns:oc=\"http://owncloud.org/ns\">
				 <d:set>
				  <d:prop>$propertyBody
				  </d:prop>
				 </d:set>
				</d:propertyupdate>";
		return self::makeDavRequest(
			$baseUrl,
			$user,
			$password,
			"PROPPATCH",
			$path,
			[],
			null,
			$xRequestId,
			$body,
			$davPathVersion,
			$type
		);
	}

	/**
	 * returns the response to listing a folder (collection)
	 *
	 * @param string|null $baseUrl
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $path
	 * @param string|null $folderDepth
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param string[] $properties
	 * @param string|null $type
	 * @param int|null $davPathVersionToUse
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public static function listFolder(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $path,
		?string $folderDepth,
		?string $spaceId = null,
		?string $xRequestId = '',
		?array $properties = null,
		?string $type = "files",
		?int $davPathVersionToUse = self::DAV_VERSION_NEW
	):ResponseInterface {
		if (!$properties) {
			$properties = [
				'd:getetag', 'd:resourcetype'
			];
		}
		return self::propfind(
			$baseUrl,
			$user,
			$password,
			$path,
			$properties,
			$xRequestId,
			$folderDepth,
			$spaceId,
			$type,
			$davPathVersionToUse
		);
	}

	/**
	 * Generates UUIDv4
	 * Example: 123e4567-e89b-12d3-a456-426614174000
	 *
	 * @return string
	 * @throws Exception
	 */
	public static function generateUUIDv4():string {
		// generate 16 bytes (128 bits) of random data or use the data passed into the function.
		$data = random_bytes(16);
		\assert(\strlen($data) == 16);

		$data[6] = \chr(\ord($data[6]) & 0x0f | 0x40); // set version to 0100
		$data[8] = \chr(\ord($data[8]) & 0x3f | 0x80); // set bits 6-7 to 10

		return vsprintf('%s%s-%s-%s-%s-%s%s%s', str_split(bin2hex($data), 4));
	}

	/**
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getSharesSpaceIdForUser(string $baseUrl, string $user, string $password, string $xRequestId): string {
		if (\array_key_exists($user, self::$spacesIdRef) && \array_key_exists("virtual", self::$spacesIdRef[$user])) {
			return self::$spacesIdRef[$user]["virtual"];
		}

		$response = GraphHelper::getMySpaces($baseUrl, $user, $password, '', $xRequestId);
		$body = HttpRequestHelper::getJsonDecodedResponseBodyContent($response);

		$spaceId = null;
		foreach ($body->value as $spaces) {
			if ($spaces->driveType === "virtual") {
				$spaceId = $spaces->id;
				break;
			}
		}
		return $spaceId;
	}

	/**
	 * fetches personal space id for provided user
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function getPersonalSpaceIdForUser(string $baseUrl, string $user, string $password, string $xRequestId):string {
		if (\array_key_exists($user, self::$spacesIdRef) && \array_key_exists("personal", self::$spacesIdRef[$user])) {
			return self::$spacesIdRef[$user]["personal"];
		}
		$trimmedBaseUrl = \trim($baseUrl, "/");
		$drivesPath = '/graph/v1.0/me/drives';
		$fullUrl = $trimmedBaseUrl . $drivesPath;
		$response = HttpRequestHelper::get(
			$fullUrl,
			$xRequestId,
			$user,
			$password
		);
		$bodyContents = $response->getBody()->getContents();
		$json = \json_decode($bodyContents);
		$personalSpaceId = '';
		if ($json === null) {
			// the graph endpoint did not give a useful answer
			// try getting the information from the webdav endpoint
			$fullUrl = "$trimmedBaseUrl/" . self::getDavPath(self::DAV_VERSION_NEW, $user);
			$response = HttpRequestHelper::sendRequest(
				$fullUrl,
				$xRequestId,
				'PROPFIND',
				$user,
				$password
			);
			// we expect to get a multipart XML response with status 207
			$status = $response->getStatusCode();
			if ($status === 401) {
				throw new SpaceNotFoundException(__METHOD__ . " Personal space not found for user " . $user);
			} elseif ($status !== 207) {
				throw new Exception(
					__METHOD__ . " webdav propfind for user $user failed with status $status - so the personal space id cannot be discovered"
				);
			}
			$responseXmlObject = HttpRequestHelper::getResponseXml(
				$response,
				__METHOD__
			);
			$xmlPart = $responseXmlObject->xpath("/d:multistatus/d:response[1]/d:propstat/d:prop/oc:id");
			if ($xmlPart === false) {
				throw new Exception(
					__METHOD__ . " oc:id not found in webdav propfind for user $user - so the personal space id cannot be discovered"
				);
			}
			$ocIdRawString = $xmlPart[0]->__toString();
			$separator = "!";
			if (\strpos($ocIdRawString, $separator) !== false) {
				// The string is not base64-encoded, because the exclamation mark is not in the base64 alphabet.
				// We expect to have a string with 2 parts separated by the exclamation mark.
				// This is the format introduced in 2022-02
				// oc:id should be something like:
				// "7464caf6-1799-103c-9046-c7b74deb5f63!7464caf6-1799-103c-9046-c7b74deb5f63"
				// There is no encoding to decode.
				$decodedId = $ocIdRawString;
			} else {
				// fall-back to assuming that the oc:id is base64-encoded
				// That is the format used before and up to 2022-02
				// This can be removed after both the edge and master branches of cs3org/reva are using the new format.
				// oc:id should be some base64 encoded string like:
				// "NzQ2NGNhZjYtMTc5OS0xMDNjLTkwNDYtYzdiNzRkZWI1ZjYzOjc0NjRjYWY2LTE3OTktMTAzYy05MDQ2LWM3Yjc0ZGViNWY2Mw=="
				// That should decode to something like:
				// "7464caf6-1799-103c-9046-c7b74deb5f63:7464caf6-1799-103c-9046-c7b74deb5f63"
				$decodedId = base64_decode($ocIdRawString);
				$separator = ":";
			}
			$ocIdParts = \explode($separator, $decodedId);
			if (\count($ocIdParts) !== 2) {
				throw new Exception(
					__METHOD__ . " the oc:id $decodedId for user $user does not have 2 parts separated by '$separator', so the personal space id cannot be discovered"
				);
			}
			$personalSpaceId = $ocIdParts[0];
		} else {
			foreach ($json->value as $spaces) {
				if ($spaces->driveType === "personal") {
					$personalSpaceId = $spaces->id;
					break;
				}
			}
		}
		if ($personalSpaceId) {
			// If env var LOG_PERSONAL_SPACE_ID is defined, then output the details of the personal space id.
			// This is a useful debugging tool to have confidence that the personal space id is found correctly.
			if (\getenv('LOG_PERSONAL_SPACE_ID') !== false) {
				echo __METHOD__ . " personal space id of user $user is $personalSpaceId\n";
			}
			self::$spacesIdRef[$user] = [];
			self::$spacesIdRef[$user]["personal"] = $personalSpaceId;
			return $personalSpaceId;
		} else {
			throw new SpaceNotFoundException(__METHOD__ . " Personal space not found for user " . $user);
		}
	}

	/**
	 * First checks if a user exists to return its space ID
	 * In case of any exception, it returns a fake space ID
	 *
	 * @param string $baseUrl
	 * @param string $user
	 * @param string $password
	 * @param string $xRequestId
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getPersonalSpaceIdForUserOrFakeIfNotFound(string $baseUrl, string $user, string $password, string $xRequestId):string {
		try {
			$spaceId = self::getPersonalSpaceIdForUser(
				$baseUrl,
				$user,
				$password,
				$xRequestId,
			);
		} catch (SpaceNotFoundException $e) {
			// if the fetch fails, and the user is not found, then a fake space id is prepared
			// this is useful for testing when the personal space is of a non-existing user
			$fakeSpaceId = self::generateUUIDv4();
			self::$spacesIdRef[$user]["personal"] = $fakeSpaceId;
			$spaceId = $fakeSpaceId;
		}
		return $spaceId;
	}

	/**
	 * sends a DAV request
	 *
	 * @param string|null $baseUrl
	 * URL of owncloud e.g. http://localhost:8080
	 * should include the subfolder if owncloud runs in a subfolder
	 * e.g. http://localhost:8080/owncloud-core
	 * @param string|null $user
	 * @param string|null $password or token when bearer auth is used
	 * @param string|null $method PUT, GET, DELETE, etc.
	 * @param string|null $path
	 * @param array|null $headers
	 * @param string|null $spaceId
	 * @param string|null $xRequestId
	 * @param string|null|resource|StreamInterface $body
	 * @param int|null $davPathVersionToUse (1|2|3)
	 * @param string|null $type of request
	 * @param string|null $sourceIpAddress to initiate the request from
	 * @param string|null $authType basic|bearer
	 * @param bool $stream Set to true to stream a response rather
	 *                     than download it all up-front.
	 * @param int|null $timeout
	 * @param Client|null $client
	 * @param array|null $urlParameter to concatenate with path
	 * @param string|null $doDavRequestAsUser run the DAV as this user, if null it is the same as $user
	 * @param bool $isGivenStep is set to true if makeDavRequest is called from a "given" step
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public static function makeDavRequest(
		?string $baseUrl,
		?string $user,
		?string $password,
		?string $method,
		?string $path,
		?array $headers,
		?string $spaceId = null,
		?string $xRequestId = '',
		$body = null,
		?int $davPathVersionToUse = self::DAV_VERSION_OLD,
		?string $type = "files",
		?string $sourceIpAddress = null,
		?string $authType = "basic",
		?bool $stream = false,
		?int $timeout = 0,
		?Client $client = null,
		?array $urlParameter = [],
		?string $doDavRequestAsUser = null,
		?bool $isGivenStep = false,
	):ResponseInterface {
		$baseUrl = self::sanitizeUrl($baseUrl, true);

		// We need to manipulate and use path as a string.
		// So ensure that it is a string to avoid any type-conversion errors.
		if ($path === null) {
			$path = "";
		}

		// get space id if testing with spaces dav
		if ($spaceId === null && $davPathVersionToUse === self::DAV_VERSION_SPACES && !\in_array($type, ["public-files", "versions"])) {
			$path = \ltrim($path, "/");
			if (\str_starts_with($path, "Shares/")) {
				$spaceId = self::getSharesSpaceIdForUser(
					$baseUrl,
					$user,
					$password,
					$xRequestId
				);
				$path = "/" . preg_replace("/^Shares\//", "", $path);
			} else {
				$spaceId = self::getPersonalSpaceIdForUserOrFakeIfNotFound(
					$baseUrl,
					$user,
					$password,
					$xRequestId
				);
			}
		}

		$suffixPath = $user;
		if ($davPathVersionToUse === self::DAV_VERSION_SPACES && !\in_array($type, ["archive", "versions", "public-files"])) {
			$suffixPath = $spaceId;
		} elseif ($type === "versions") {
			// $path is file-id in case of versions
			$suffixPath = $path;
		}

		$davPath = self::getDavPath($davPathVersionToUse, $suffixPath, $type);

		//replace %, # and ? and in the path, Guzzle will not encode them
		$urlSpecialChar = [['%', '#', '?'], ['%25', '%23', '%3F']];
		$path = \str_replace($urlSpecialChar[0], $urlSpecialChar[1], $path);

		if (!empty($urlParameter)) {
			$urlParameter = \http_build_query($urlParameter, '', '&');
			$path .= '?' . $urlParameter;
		}
		$fullUrl = self::sanitizeUrl("$baseUrl/$davPath");
		// NOTE: no need to append path for archive and versions endpoints
		if (!\in_array($type, ["archive", "versions"])) {
			$fullUrl .= "/" . \ltrim($path, "/");
		}

		if ($authType === 'bearer') {
			$headers['Authorization'] = 'Bearer ' . $password;
			$user = null;
			$password = null;
		}
		if ($type === "public-files") {
			if ($password === null || $password === "") {
				$user = null;
			} else {
				$user = "public";
			}
		}
		$config = null;
		if ($sourceIpAddress !== null) {
			$config = [ 'curl' => [ CURLOPT_INTERFACE => $sourceIpAddress ]];
		}

		if ($headers !== null) {
			foreach ($headers as $key => $value) {
				//? and # need to be encoded in the Destination URL
				if ($key === "Destination") {
					$headers[$key] = \str_replace(
						$urlSpecialChar[0],
						$urlSpecialChar[1],
						$value
					);
					break;
				}
			}
		}

		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$xRequestId,
			$method,
			$doDavRequestAsUser ?? $user,
			$password,
			$headers,
			$body,
			$config,
			null,
			$stream,
			$timeout,
			$client,
			$isGivenStep
		);
	}

	/**
	 * get the dav path
	 *
	 * @param int $davPathVersion (1|2|3)
	 * @param string|null $userOrItemIdOrSpaceIdOrToken 'user' or 'file-id' or 'space-id' or 'public-token'
	 * @param string|null $type
	 *
	 * @return string
	 */
	public static function getDavPath(
		int $davPathVersion,
		?string $userOrItemIdOrSpaceIdOrToken = null,
		?string $type = "files"
	):string {
		switch ($type) {
			case 'archive':
				return self::prefixRemotePhp("dav/archive/$userOrItemIdOrSpaceIdOrToken/files");
			case 'versions':
				return self::prefixRemotePhp("dav/meta/$userOrItemIdOrSpaceIdOrToken/v");
			case 'comments':
				return self::prefixRemotePhp("dav/comments/files");
			default:
				break;
		}

		if ($davPathVersion === self::DAV_VERSION_SPACES) {
			if ($type === "trash-bin") {
				if ($userOrItemIdOrSpaceIdOrToken === null) {
					throw new InvalidArgumentException("Space ID is required for trash-bin endpoint");
				}
				return self::prefixRemotePhp("dav/spaces/trash-bin/$userOrItemIdOrSpaceIdOrToken");
			} elseif ($type === "public-files") {
				// spaces DAV path doesn't have own public-files endpoint
				return self::prefixRemotePhp("dav/public-files/$userOrItemIdOrSpaceIdOrToken");
			}
			// return spaces root path if spaceid is null
			// REPORT request uses spaces root path
			if ($userOrItemIdOrSpaceIdOrToken === null) {
				return self::prefixRemotePhp("dav/spaces");
			}
			return self::prefixRemotePhp("dav/spaces/$userOrItemIdOrSpaceIdOrToken");
		} else {
			if ($type === "trash-bin") {
				// Since there is no trash bin endpoint for old dav version,
				// new dav version's endpoint is used here.
				return self::prefixRemotePhp("dav/trash-bin/$userOrItemIdOrSpaceIdOrToken");
			}
			if ($davPathVersion === self::DAV_VERSION_OLD) {
				return self::prefixRemotePhp("webdav");
			} elseif ($davPathVersion === self::DAV_VERSION_NEW) {
				if ($type === "files") {
					return self::prefixRemotePhp("dav/files/$userOrItemIdOrSpaceIdOrToken");
				} elseif ($type === "public-files") {
					return self::prefixRemotePhp("dav/public-files/$userOrItemIdOrSpaceIdOrToken");
				}
				return self::prefixRemotePhp("dav");
			}
			throw new InvalidArgumentException("Invalid DAV path: $davPathVersion");
		}
	}

	/**
	 * make sure there are no double-slashes in the URL
	 *
	 * @param string|null $url
	 * @param bool|null $trailingSlash forces a trailing slash
	 *
	 * @return string
	 */
	public static function sanitizeUrl(?string $url, ?bool $trailingSlash = false):string {
		if ($trailingSlash === true) {
			$url = $url . "/";
		} else {
			$url = \rtrim($url, "/");
		}
		return \preg_replace("/([^:]\/)\/+/", '$1', $url);
	}

	/**
	 * Decides if the proposed dav version and chunking version are
	 * a valid combination.
	 * If no chunkingVersion is specified, then any dav version is valid.
	 * If a chunkingVersion is specified, then it has to match the dav version.
	 * Note: in future, the dav and chunking versions might or might not
	 * move together and/or be supported together. So a more complex
	 * matrix could be needed here.
	 *
	 * @param string|int $davPathVersion
	 * @param string|int|null $chunkingVersion
	 *
	 * @return boolean is this a valid combination
	 */
	public static function isValidDavChunkingCombination(
		$davPathVersion,
		$chunkingVersion
	): bool {
		if ($davPathVersion === self::DAV_VERSION_SPACES) {
			// allow only old chunking version when using the spaces dav
			return $chunkingVersion === 1;
		}
		return (
			($chunkingVersion === 'no' || $chunkingVersion === null) ||
			($davPathVersion === $chunkingVersion)
		);
	}

	/**
	 * get Mtime of File in a public link share
	 *
	 * @param string|null $baseUrl
	 * @param string|null $fileName
	 * @param string|null $token
	 * @param string|null $xRequestId
	 * @param int|null $davVersionToUse
	 *
	 * @return string
	 * @throws Exception|GuzzleException
	 */
	public static function getMtimeOfFileInPublicLinkShare(
		?string $baseUrl,
		?string $fileName,
		?string $token,
		?string $xRequestId = '',
		?int $davVersionToUse = self::DAV_VERSION_NEW
	):string {
		$response = self::propfind(
			$baseUrl,
			null,
			null,
			"$token/$fileName",
			['d:getlastmodified'],
			$xRequestId,
			'1',
			null,
			"public-files",
			$davVersionToUse
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:getlastmodified");

		return $xmlPart[0]->__toString();
	}

	/**
	 * get Mtime of a resource
	 *
	 * @param string|null $user
	 * @param string|null $password
	 * @param string|null $baseUrl
	 * @param string|null $resource
	 * @param string|null $xRequestId
	 * @param int|null $davPathVersionToUse
	 * @param string|null $spaceId
	 *
	 * @return string
	 * @throws Exception
	 * @throws GuzzleException
	 */
	public static function getMtimeOfResource(
		?string $user,
		?string $password,
		?string $baseUrl,
		?string $resource,
		?string $xRequestId = '',
		?int $davPathVersionToUse = self::DAV_VERSION_NEW,
		?string $spaceId = null,
	):string {
		$response = self::propfind(
			$baseUrl,
			$user,
			$password,
			$resource,
			["d:getlastmodified"],
			$xRequestId,
			"0",
			$spaceId,
			"files",
			$davPathVersionToUse
		);
		$responseXmlObject = HttpRequestHelper::getResponseXml(
			$response,
			__METHOD__
		);
		$xmlPart = $responseXmlObject->xpath("//d:getlastmodified");
		Assert::assertArrayHasKey(
			0,
			$xmlPart,
			__METHOD__ . " XML part does not have key 0. Expected a value at index 0 of 'xmlPart' but, found: " . json_encode($xmlPart)
		);
		$mtime = new DateTime($xmlPart[0]->__toString());
		return $mtime->format('U');
	}
}
