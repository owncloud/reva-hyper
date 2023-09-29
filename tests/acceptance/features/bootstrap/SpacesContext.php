<?php

declare(strict_types=1);
/**
 * ownCloud
 *
 * @author Michael Barz <mbarz@owncloud.com>
 * @copyright Copyright (c) 2021 Michael Barz mbarz@owncloud.com
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

use Behat\Behat\Context\Context;
use Behat\Behat\Hook\Scope\BeforeScenarioScope;
use Behat\Gherkin\Node\PyStringNode;
use Behat\Gherkin\Node\TableNode;
use GuzzleHttp\Exception\GuzzleException;
use Psr\Http\Message\ResponseInterface;
use TestHelpers\HttpRequestHelper;
use TestHelpers\WebDavHelper;
use TestHelpers\SetupHelper;
use TestHelpers\GraphHelper;
use PHPUnit\Framework\Assert;

require_once 'bootstrap.php';

/**
 * Context for ocis spaces specific steps
 */
class SpacesContext implements Context {
	private FeatureContext $featureContext;
	private OCSContext $ocsContext;
	private TrashbinContext $trashbinContext;
	private WebDavPropertiesContext $webDavPropertiesContext;
	private FavoritesContext $favoritesContext;
	private ChecksumContext $checksumContext;
	private FilesVersionsContext $filesVersionsContext;
	private string $baseUrl;

	/**
	 * key is space name and value is the username that created the space
	 */
	private array $createdSpaces;
	private string $ocsApiUrl = '/ocs/v2.php/apps/files_sharing/api/v1/shares';
	private string $davSpacesUrl = '/remote.php/dav/spaces/';

	/**
	 * @var array map with user as key, spaces and file etags as value
	 * @example
	 * [
	 *   "user1" => [
	 *     "Personal" => [
	 *       "file1.txt": "etag1",
	 *     ],
	 *     "Shares" => []
	 *   ],
	 *   "user2" => [
	 *     ...
	 *   ]
	 * ]
	 */
	private array $storedEtags = [];

	/**
	 * @param string $spaceName
	 *
	 * @return string name of the user that created the space
	 * @throws Exception
	 */
	public function getSpaceCreator(string $spaceName): string {
		if (!\array_key_exists($spaceName, $this->createdSpaces)) {
			throw new Exception(__METHOD__ . " space '$spaceName' has not been created in this scenario");
		}
		return $this->createdSpaces[$spaceName];
	}

	/**
	 * @param string $spaceName
	 * @param string $spaceCreator
	 *
	 * @return void
	 */
	public function setSpaceCreator(string $spaceName, string $spaceCreator): void {
		$this->createdSpaces[$spaceName] = $spaceCreator;
	}

	private array $availableSpaces = [];

	/**
	 * @return array
	 */
	public function getAvailableSpaces(): array {
		return $this->availableSpaces;
	}

	/**
	 * @param array $availableSpaces
	 *
	 * @return void
	 */
	public function setAvailableSpaces(array $availableSpaces): void {
		$this->availableSpaces = $availableSpaces;
	}

	/**
	 * response content parsed from XML to an array
	 *
	 * @var array
	 */
	private array $responseXml = [];

	/**
	 * @return array
	 */
	public function getResponseXml(): array {
		return $this->responseXml;
	}

	/**
	 * @param array $responseXml
	 *
	 * @return void
	 */
	public function setResponseXml(array $responseXml): void {
		$this->responseXml = $responseXml;
	}

	/**
	 * Get SpaceId by Name
	 *
	 * @param $name string
	 *
	 * @return string
	 *
	 * @throws Exception
	 */
	public function getSpaceIdByNameFromResponse(string $name): string {
		$space = $this->getSpaceByNameFromResponse($name);
		Assert::assertIsArray($space, "Space with name $name not found");
		if (!isset($space["id"])) {
			throw new Exception(__METHOD__ . " space with name $name not found");
		}
		return $space["id"];
	}

	/**
	 * Get Space Array by name
	 *
	 * @param string $name
	 *
	 * @return array
	 *
	 * @throws Exception
	 */
	public function getSpaceByNameFromResponse(string $name): array {
		$response = json_decode((string)$this->featureContext->getResponse()->getBody(), true, 512, JSON_THROW_ON_ERROR);
		$spaceAsArray = $response;
		if (isset($response['name']) && $response['name'] === $name) {
			return $response;
		}
		if (isset($spaceAsArray["value"])) {
			foreach ($spaceAsArray["value"] as $spaceCandidate) {
				if ($spaceCandidate['name'] === $name) {
					return $spaceCandidate;
				}
			}
		}
		return [];
	}

	/**
	 * The method finds available spaces to the user and returns the space by spaceName
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getSpaceByName(string $user, string $spaceName): array {
		if ($spaceName === "Personal") {
			$spaceName = $this->featureContext->getUserDisplayName($user);
		}
		if (strtolower($user) === 'admin') {
			$this->theUserListsAllAvailableSpacesUsingTheGraphApi($user);
		} else {
			$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		}
		$spaces = $this->getAvailableSpaces();
		Assert::assertArrayHasKey($spaceName, $spaces, "Space with name $spaceName for user $user not found");
		Assert::assertNotEmpty($spaces[$spaceName]["root"]["webDavUrl"], "WebDavUrl for space with name $spaceName for user $user not found");
		return $spaces[$spaceName];
	}

	/**
	 * The method finds available spaces to the user and returns the spaceId by spaceName
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getSpaceIdByName(string $user, string $spaceName): string {
		$space = $this->getSpaceByName($user, $spaceName);
		return $space["id"];
	}

	/**
	 * This method sets space id by Space Name
	 * This is currently used to set space id from ocis in core so that we can reuse available resource (code) and avoid duplication
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function setSpaceIDByName(string $user, string $spaceName): void {
		$space = $this->getSpaceByName($user, $spaceName);
		Assert::assertIsArray($space, "Space with name $spaceName not found");
		Assert::assertNotEmpty($space["root"]["webDavUrl"], "WebDavUrl for space with name $spaceName not found");
		WebDavHelper::$SPACE_ID_FROM_OCIS = $space['id'];
	}

	/**
	 * The method finds available spaces to the manager user and returns the space by spaceName
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getSpaceByNameManager(string $user, string $spaceName): array {
		$this->theUserListsAllAvailableSpacesUsingTheGraphApi($user);

		$spaces = $this->getAvailableSpaces();
		Assert::assertArrayHasKey($spaceName, $spaces, "Space with name '$spaceName' for user '$user' not found");
		Assert::assertIsArray($spaces[$spaceName], "Data for space with name '$spaceName' for user '$user' not found");
		Assert::assertNotEmpty($spaces[$spaceName]["root"]["webDavUrl"], "WebDavUrl for space with name '$spaceName' for user '$user' not found");

		return $spaces[$spaceName];
	}

	/**
	 * The method finds file by fileName and spaceName and returns data of file which contains in responseHeader
	 * fileName contains the path, if the file is in the folder
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function getFileData(string $user, string $spaceName, string $fileName): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$fullUrl = $this->baseUrl . $this->davSpacesUrl . $space["id"] . "/" . $fileName;

		return HttpRequestHelper::get(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			[],
			"{}"
		);
	}

	/**
	 * The method returns fileId
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getFileId(string $user, string $spaceName, string $fileName): string {
		$fileData = $this->getFileData($user, $spaceName, $fileName)->getHeaders();
		return $fileData["Oc-Fileid"][0];
	}

	/**
	 * The method returns "fileid" from the PROPFIND response
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $folderName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getResourceId(string $user, string $spaceName, string $folderName): string {
		$space = $this->getSpaceByName($user, $spaceName);
		// For a level 1 folder, the parent is space so $folderName = ''
		if ($folderName === $space["name"]) {
			$folderName = '';
		}
		$fullUrl = $this->baseUrl . $this->davSpacesUrl . $space["id"] . "/" . $folderName;
		$response = HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'PROPFIND',
			$user,
			$this->featureContext->getPasswordForUser($user),
			['Depth' => '0'],
		);
		$responseArray = json_decode(json_encode($this->featureContext->getResponseXml($response)->xpath("//d:response/d:propstat/d:prop/oc:fileid")), true, 512, JSON_THROW_ON_ERROR);
		Assert::assertNotEmpty($responseArray, "the PROPFIND response for $folderName is empty");
		return $responseArray[0][0];
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return string
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function getPrivateLink(string $user, string $spaceName): string {
		$this->setSpaceIDByName($user, $spaceName);
		$response = WebDavHelper::propfind(
			$this->featureContext->getBaseUrl(),
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			"",
			['oc:privatelink'],
			$this->featureContext->getStepLineRef(),
			"0",
			"files",
			WebDavHelper::DAV_VERSION_SPACES
		);
		$responseArray = json_decode(json_encode($this->featureContext->getResponseXml($response)->xpath("//d:response/d:propstat/d:prop/oc:privatelink")), true, 512, JSON_THROW_ON_ERROR);
		Assert::assertNotEmpty($responseArray, "the PROPFIND response for $spaceName is empty");
		return $responseArray[0][0];
	}

	/**
	 * The method returns eTag
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileName
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function getETag(string $user, string $spaceName, string $fileName): string {
		$fileData = $this->getFileData($user, $spaceName, $fileName)->getHeaders();
		return \str_replace('"', '\"', $fileData["Etag"][0]);
	}

	/**
	 * @BeforeScenario
	 *
	 * @param BeforeScenarioScope $scope
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function setUpScenario(BeforeScenarioScope $scope): void {
		// Get the environment
		$environment = $scope->getEnvironment();
		// register new context
		$this->trashbinContext = new TrashbinContext();
		$this->webDavPropertiesContext = new WebDavPropertiesContext();
		$this->favoritesContext = new FavoritesContext();
		$this->checksumContext = new ChecksumContext();
		$this->filesVersionsContext = new FilesVersionsContext();
		$environment->registerContext($this->trashbinContext);
		$environment->registerContext($this->webDavPropertiesContext);
		$environment->registerContext($this->favoritesContext);
		$environment->registerContext($this->checksumContext);
		$environment->registerContext($this->filesVersionsContext);
		// Get all the contexts you need in this context
		$this->featureContext = $environment->getContext('FeatureContext');
		$this->ocsContext = $environment->getContext('OCSContext');
		// Run the BeforeScenario function in OCSContext to set it up correctly
		$this->ocsContext->before($scope);
		$this->baseUrl = \trim($this->featureContext->getBaseUrl(), "/");
		SetupHelper::init(
			$this->featureContext->getAdminUsername(),
			$this->featureContext->getAdminPassword(),
			$this->baseUrl,
			$this->featureContext->getOcPath()
		);
	}

	/**
	 * @AfterScenario
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function cleanDataAfterTests(): void {
		$this->deleteAllProjectSpaces();
	}

	/**
	 * the admin user first disables and then deletes spaces
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function deleteAllProjectSpaces(): void {
		$query = "\$filter=driveType eq project";
		$userAdmin = $this->featureContext->getAdminUsername();

		$this->theUserListsAllAvailableSpacesUsingTheGraphApi(
			$userAdmin,
			$query
		);
		$drives = $this->getAvailableSpaces();

		foreach ($drives as $value) {
			if (!\array_key_exists("deleted", $value["root"])) {
				GraphHelper::disableSpace(
					$this->featureContext->getBaseUrl(),
					$userAdmin,
					$this->featureContext->getPasswordForUser($userAdmin),
					$value["id"],
					$this->featureContext->getStepLineRef()
				);
			}
			GraphHelper::deleteSpace(
				$this->featureContext->getBaseUrl(),
				$userAdmin,
				$this->featureContext->getPasswordForUser($userAdmin),
				$value["id"],
				$this->featureContext->getStepLineRef()
			);
		}
	}

	/**
	 * Send POST Request to url
	 *
	 * @param string $fullUrl
	 * @param string $user
	 * @param string $password
	 * @param mixed $body
	 * @param string $xRequestId
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function sendPostRequestToUrl(
		string $fullUrl,
		string $user,
		string $password,
		$body,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		return HttpRequestHelper::post($fullUrl, $xRequestId, $user, $password, $headers, $body);
	}

	/**
	 * Send Graph Create Folder Request
	 *
	 * @param  string $fullUrl
	 * @param  string $method
	 * @param  string $user
	 * @param  string $password
	 * @param  string $xRequestId
	 * @param  array  $headers
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function sendCreateFolderRequest(
		string $fullUrl,
		string $method,
		string $user,
		string $password,
		string $xRequestId = '',
		array $headers = []
	): ResponseInterface {
		return HttpRequestHelper::sendRequest($fullUrl, $xRequestId, $method, $user, $password, $headers);
	}

	/**
	 * @When /^user "([^"]*)" lists all available spaces via the Graph API$/
	 * @When /^user "([^"]*)" lists all available spaces via the Graph API with query "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $query
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserListsAllHisAvailableSpacesUsingTheGraphApi(string $user, string $query = ''): void {
		$this->featureContext->setResponse(
			GraphHelper::getMySpaces(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				"?" . $query,
				$this->featureContext->getStepLineRef()
			)
		);
		$this->rememberTheAvailableSpaces();
	}

	/**
	 * The method is used on the administration setting tab, which only the Admin user and the Space admin user have access to
	 *
	 * @When /^user "([^"]*)" lists all spaces via the Graph API$/
	 * @When /^user "([^"]*)" lists all spaces via the Graph API with query "([^"]*)"$/
	 * @When /^user "([^"]*)" tries to list all spaces via the Graph API$/
	 *
	 * @param string $user
	 * @param string $query
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserListsAllAvailableSpacesUsingTheGraphApi(string $user, string $query = ''): void {
		$this->featureContext->setResponse(
			GraphHelper::getAllSpaces(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				"?" . $query,
				$this->featureContext->getStepLineRef()
			)
		);
		$this->rememberTheAvailableSpaces();
	}

	/**
	 * @When /^user "([^"]*)" looks up the single space "([^"]*)" via the Graph API by using its id$/
	 * @When /^user "([^"]*)" tries to look up the single space "([^"]*)" owned by the user "([^"]*)" by using its id$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $ownerUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theUserLooksUpTheSingleSpaceUsingTheGraphApiByUsingItsId(string $user, string $spaceName, string $ownerUser = ''): void {
		$space = $this->getSpaceByName(($ownerUser !== "") ? $ownerUser : $user, $spaceName);
		Assert::assertIsArray($space);
		Assert::assertNotEmpty($space["id"]);
		Assert::assertNotEmpty($space["root"]["webDavUrl"]);
		$this->featureContext->setResponse(
			GraphHelper::getSingleSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$space["id"],
				'',
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:creates|tries to create) a space "([^"]*)" of type "([^"]*)" with quota "([^"]*)" using the Graph API$/
	 * @When /^user "([^"]*)" (?:creates|tries to create) a space "([^"]*)" of type "([^"]*)" with the default quota using the Graph API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $spaceType
	 * @param int|null $quota
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserCreatesASpaceWithQuotaUsingTheGraphApi(
		string $user,
		string $spaceName,
		string $spaceType,
		?int $quota = null
	): void {
		$space = ["Name" => $spaceName, "driveType" => $spaceType, "quota" => ["total" => $quota]];
		$body = json_encode($space);
		$this->featureContext->setResponse(
			GraphHelper::createSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$body,
				$this->featureContext->getStepLineRef()
			)
		);
		$this->setSpaceCreator($spaceName, $user);
	}

	/**
	 * Remember the available Spaces
	 *
	 * @param ResponseInterface|null $response
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function rememberTheAvailableSpaces(?ResponseInterface $response = null): void {
		if ($response) {
			$rawBody =  $response->getBody()->getContents();
		} else {
			$rawBody =  $this->featureContext->getResponse()->getBody()->getContents();
		}
		$drives = json_decode($rawBody, true, 512, JSON_THROW_ON_ERROR);
		if (isset($drives["value"])) {
			$drives = $drives["value"];
		}

		$spaces = [];
		foreach ($drives as $drive) {
			$spaces[$drive["name"]] = $drive;
		}
		$this->setAvailableSpaces($spaces);
	}

	/**
	 * @When /^user "([^"]*)" lists the content of the space with the name "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $foldersPath
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theUserListsTheContentOfAPersonalSpaceRootUsingTheWebDAvApi(
		string $user,
		string $spaceName,
		string $foldersPath = ''
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->setResponse(
			WebDavHelper::propfind(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$foldersPath,
				[],
				$this->featureContext->getStepLineRef(),
				'infinity',
				'files',
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" sends PATCH request to the space "([^"]*)" of user "([^"]*)" with data "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $owner
	 * @param string $data
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userSendsPatchRequestToTheSpaceOfUserWithData(string $user, string $spaceName, string $owner, string $data): void {
		$space = $this->getSpaceByName($owner, $spaceName);
		Assert::assertIsArray($space);
		Assert::assertNotEmpty($spaceId = $space["id"]);
		$url = GraphHelper::getFullUrl($this->featureContext->getBaseUrl(), 'drives/' . $spaceId);
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$url,
				$this->featureContext->getStepLineRef(),
				"PATCH",
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				null,
				$data
			)
		);
	}

	/**
	 * @Then /^the (?:propfind|search) result of the space should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function thePropfindResultShouldContainEntries(
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$this->featureContext->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
		);
		WebDavHelper::$SPACE_ID_FROM_OCIS = '';
	}

	/**
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $user
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userTheSpaceShouldContainEntries(
		string $user,
		string $spaceName,
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$this->theUserListsTheContentOfAPersonalSpaceRootUsingTheWebDAvApi(
			$user,
			$spaceName
		);
		WebDavHelper::$SPACE_ID_FROM_OCIS = $space['id'];
		$this->featureContext->propfindResultShouldContainEntries($shouldOrNot, $expectedFiles, $user, 'PROPFIND');
		WebDavHelper::$SPACE_ID_FROM_OCIS = '';
	}

	/**
	 * @Then /^for user "([^"]*)" folder "([^"]*)" of the space "([^"]*)" should (not|)\s?contain these (?:files|entries):$/
	 *
	 * @param string    $user
	 * @param string    $folderPath
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param TableNode $expectedFiles
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function folderOfTheSpaceShouldContainEntries(
		string $user,
		string $folderPath,
		string $spaceName,
		string $shouldOrNot,
		TableNode $expectedFiles
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$this->theUserListsTheContentOfAPersonalSpaceRootUsingTheWebDAvApi(
			$user,
			$spaceName,
			$folderPath
		);
		WebDavHelper::$SPACE_ID_FROM_OCIS = $space['id'];
		$this->featureContext->propfindResultShouldContainEntries(
			$shouldOrNot,
			$expectedFiles,
			$this->featureContext->getActualUsername($user),
			'PROPFIND',
			$folderPath
		);
		WebDavHelper::$SPACE_ID_FROM_OCIS = '';
	}

	/**
	 * @Then /^for user "([^"]*)" the content of the file "([^"]*)" of the space "([^"]*)" should be "([^"]*)"$/
	 *
	 * @param string    $user
	 * @param string    $file
	 * @param string    $spaceName
	 * @param string    $fileContent
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function checkFileContent(
		string $user,
		string $file,
		string $spaceName,
		string $fileContent
	): void {
		$actualFileContent = $this->getFileData($user, $spaceName, $file)->getBody()->getContents();
		Assert::assertEquals($fileContent, $actualFileContent, "$file does not contain $fileContent");
	}

	/**
	 * @Then /^the JSON response should contain space called "([^"]*)" (?:|(?:owned by|granted to) "([^"]*)" )(?:|(?:with description file|with space image) "([^"]*)" )and match$/
	 *
	 * @param string $spaceName
	 * @param string|null $userName
	 * @param string|null $fileName
	 * @param PyStringNode|null $schemaString
	 *
	 * @return void
	 */
	public function theJsonDataFromLastResponseShouldMatch(
		string $spaceName,
		?string $userName = null,
		?string $fileName = null,
		?PyStringNode $schemaString = null
	): void {
		Assert::assertNotNull($schemaString, 'schema is not valid JSON');

		if (isset($this->featureContext->getJsonDecodedResponseBodyContent()->value)) {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent()->value;
			foreach ($responseBody as $value) {
				if (isset($value->name) && $value->name === $spaceName) {
					$responseBody = $value;
					break;
				}
			}
		} else {
			$responseBody = $this->featureContext->getJsonDecodedResponseBodyContent();
		}

		// substitute the value here
		if (\gettype($schemaString) !== 'string') {
			$schemaString = $schemaString->getRaw();
		}
		$schemaString = $this->featureContext->substituteInLineCodes(
			$schemaString,
			$this->featureContext->getCurrentUser(),
			[],
			[
				[
					"code" => "%space_id%",
					"function" =>
						[$this, "getSpaceIdByNameFromResponse"],
					"parameter" => [$spaceName]
				],
				[
					"code" => "%file_id%",
					"function" =>
						[$this, "getFileId"],
					"parameter" => [$userName, $spaceName, $fileName]
				],
				[
					"code" => "%eTag%",
					"function" =>
						[$this, "getETag"],
					"parameter" => [$userName, $spaceName, $fileName]
				],
			],
			null,
			$userName,
		);

		$this->featureContext->assertJsonDocumentMatchesSchema(
			$responseBody,
			$this->featureContext->getJSONSchema($schemaString)
		);
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to "([^"]*)" with role "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $grantedUser
	 * @param string $role
	 *
	 * @return void
	 * @throws Exception|GuzzleException
	 */
	public function checkPermissionsInResponse(
		string $user,
		string $spaceName,
		string $grantedUser,
		string $role
	): void {
		$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
		Assert::assertIsArray($spaceAsArray = $this->getSpaceByNameFromResponse($spaceName), "No space with name $spaceName found");
		$permissions = $spaceAsArray["root"]["permissions"];
		$userId = $this->featureContext->getUserIdByUserName($grantedUser);

		$userRole = "";
		foreach ($permissions as $permission) {
			foreach ($permission["grantedToIdentities"] as $grantedToIdentities) {
				if ($grantedToIdentities["user"]["id"] === $userId) {
					$userRole = $permission["roles"][0];
				}
			}
		}
		Assert::assertEquals($userRole, $role, "the user $grantedUser with the role $role could not be found");
	}

	/**
	 * @Then /^the json responded should not contain a space with name "([^"]*)"$/
	 *
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function jsonRespondedShouldNotContain(
		string $spaceName
	): void {
		Assert::assertEmpty($this->getSpaceByNameFromResponse($spaceName), "space $spaceName should not be available for a user");
	}

	/**
	 * @Then /^the user "([^"]*)" should (not |)have a space called "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $shouldOrNot
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userShouldNotHaveSpace(
		string $user,
		string $shouldOrNot,
		string $spaceName
	): void {
		$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
		if (\trim($shouldOrNot) === "not") {
			$this->jsonRespondedShouldNotContain($spaceName);
		} else {
			Assert::assertNotEmpty($this->getSpaceByNameFromResponse($spaceName), "space '$spaceName' should be available for a user '$user' but not found");
		}
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space "([^"]*)" in the disable state$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return mixed
	 * @throws Exception
	 */
	public function theUserShouldHaveASpaceInTheDisableState(
		string $user,
		string $spaceName
	): void {
		$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
		$response = json_decode((string)$this->featureContext->getResponse()->getBody(), true, 512, JSON_THROW_ON_ERROR);
		if (isset($response["value"])) {
			foreach ($response["value"] as $spaceCandidate) {
				if ($spaceCandidate['name'] === $spaceName) {
					if ($spaceCandidate['root']['deleted']['state'] !== 'trashed') {
						throw new \Exception(
							"space $spaceName should be in disable state but it's not "
						);
					}
					return;
				}
			}
		}
		throw new \Exception("space '$spaceName' should be available for a user '$user' but not found");
	}

	/**
	 * @Then /^the json responded should (not|only|)\s?contain spaces of type "([^"]*)"$/
	 *
	 * @param string $onlyOrNot (not|only|)
	 * @param string $type
	 *
	 * @return void
	 * @throws Exception
	 */
	public function jsonRespondedShouldNotContainSpaceType(
		string $onlyOrNot,
		string $type
	): void {
		Assert::assertNotEmpty(
			$spaces = json_decode(
				(string)$this->featureContext
					->getResponse()->getBody(),
				true,
				512,
				JSON_THROW_ON_ERROR
			)
		);
		$matches = [];
		foreach ($spaces["value"] as $space) {
			if ($onlyOrNot === "not") {
				Assert::assertNotEquals($space["driveType"], $type);
			}
			if ($onlyOrNot === "only") {
				Assert::assertEquals($space["driveType"], $type);
			}
			if ($onlyOrNot === "" && $space["driveType"] === $type) {
				$matches[] = $space;
			}
		}
		if ($onlyOrNot === "") {
			Assert::assertNotEmpty($matches);
		}
	}

	/**
	 * Verify that the tableNode contains expected number of columns
	 *
	 * @param TableNode $table
	 * @param int $count
	 *
	 * @return void
	 *
	 * @throws Exception
	 */
	public function verifyTableNodeColumnsCount(
		TableNode $table,
		int $count
	): void {
		if (\count($table->getRows()) < 1) {
			throw new Exception("Table should have at least one row.");
		}
		$rowCount = \count($table->getRows()[0]);
		if ($count !== $rowCount) {
			throw new Exception("Table expected to have $count rows but found $rowCount");
		}
	}

	/**
	 * Escapes the given string for
	 * 1. Space --> %20
	 * 2. Opening Small Bracket --> %28
	 * 3. Closing Small Bracket --> %29
	 *
	 * @param string $path - File path to parse
	 *
	 * @return string
	 */
	public function escapePath(string $path): string {
		return \str_replace([" ", "(", ")"], ["%20", "%28", "%29"], $path);
	}

	/**
	 * @When /^user "([^"]*)" creates a (?:folder|subfolder) "([^"]*)" in space "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserCreatesAFolderUsingTheGraphApi(
		string $user,
		string $folder,
		string $spaceName
	): void {
		$exploded = explode('/', $folder);
		$path = '';
		for ($i = 0; $i < \count($exploded); $i++) {
			$path = $path . $exploded[$i] . '/';
			$this->theUserCreatesAFolderToAnotherOwnerSpaceUsingTheGraphApi($user, $path, $spaceName);
		}
	}

	/**
	 * @When /^user "([^"]*)" tries to create subfolder "([^"]*)" in a nonexistent folder of the space "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $subfolder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserTriesToCreateASubFolderUsingTheGraphApi(
		string $user,
		string $subfolder,
		string $spaceName
	): void {
		$this->theUserCreatesAFolderToAnotherOwnerSpaceUsingTheGraphApi($user, $subfolder, $spaceName);
	}

	/**
	 * @Given /^user "([^"]*)" has created a (?:folder|subfolder) "([^"]*)" in space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserHasCreateAFolderUsingTheGraphApi(
		string $user,
		string $folder,
		string $spaceName
	): void {
		$this->theUserCreatesAFolderUsingTheGraphApi($user, $folder, $spaceName);

		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201"
		);
	}

	/**
	 * @When /^user "([^"]*)" creates a folder "([^"]*)" in space "([^"]*)" owned by the user "([^"]*)" using the WebDav Api$/
	 *
	 * @param string $user
	 * @param string $folder
	 * @param string $spaceName
	 * @param string $ownerUser
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserCreatesAFolderToAnotherOwnerSpaceUsingTheGraphApi(
		string $user,
		string $folder,
		string $spaceName,
		string $ownerUser = ''
	): void {
		if ($ownerUser === '') {
			$ownerUser = $user;
		}
		$this->setSpaceIDByName($ownerUser, $spaceName);
		$this->featureContext->userCreatesFolder($user, $folder);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file inside space "([^"]*)" with content "([^"]*)" to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $content
	 * @param string $destination
	 *
	 * @return string[]
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsAFileToSpace(
		string $user,
		string $spaceName,
		string $content,
		string $destination
	): array {
		$this->setSpaceIDByName($user, $spaceName);
		return $this->featureContext->uploadFileWithContent($user, $content, $destination);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $source
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsALocalFileToSpace(
		string $user,
		string $source,
		string $destination,
		string $spaceName
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->userUploadsAFileTo($user, $source, $destination);
	}

	/**
	 * @When /^user "([^"]*)" uploads a file inside space "([^"]*)" owned by the user "([^"]*)" with content "([^"]*)" to "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $ownerUser
	 * @param string $content
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserUploadsAFileToAnotherOwnerSpace(
		string $user,
		string $spaceName,
		string $ownerUser,
		string $content,
		string $destination
	): void {
		$this->setSpaceIDByName($ownerUser, $spaceName);
		$this->featureContext->uploadFileWithContent($user, $content, $destination);
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param array $bodyData
	 * @param string $owner
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSpace(
		string $user,
		string $spaceName,
		array $bodyData,
		string $owner = ''
	): ResponseInterface {
		if ($spaceName === "non-existing") {
			// check sending invalid data
			$spaceId = "39c49dd3-1f24-4687-97d1-42df43f71713";
		} else {
			$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
			$spaceId = $space["id"];
		}

		$body = json_encode($bodyData, JSON_THROW_ON_ERROR);

		return GraphHelper::updateSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$spaceId,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the name of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the name of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newName
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceName(
		string $user,
		string $spaceName,
		string $newName,
		string $owner = ''
	): void {
		$bodyData = ["Name" => $newName];
		$this->featureContext->setResponse($this->updateSpace($user, $spaceName, $bodyData, $owner));
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the description of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the description of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newDescription
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSpaceDescription(
		string $user,
		string $spaceName,
		string $newDescription,
		string $owner = ''
	): void {
		$bodyData = ["description" => $newDescription];
		$this->featureContext->setResponse($this->updateSpace($user, $spaceName, $bodyData, $owner));
	}

	/**
	 * @Given /^user "([^"]*)" has changed the description of the "([^"]*)" space to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $newDescription
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function userHasChangedDescription(
		string $user,
		string $spaceName,
		string $newDescription
	): void {
		$this->updateSpaceDescription($user, $spaceName, $newDescription);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @When /^user "([^"]*)" (?:changes|tries to change) the quota of the "([^"]*)" space to "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:changes|tries to change) the quota of the "([^"]*)" space to "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param int $newQuota
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceQuota(
		string $user,
		string $spaceName,
		int $newQuota,
		string $owner = ''
	): void {
		$bodyData = ["quota" => ["total" => $newQuota]];
		$this->featureContext->setResponse($this->updateSpace($user, $spaceName, $bodyData, $owner));
	}

	/**
	 * @Given /^user "([^"]*)" has changed the quota of the personal space of "([^"]*)" space to "([^"]*)"$/
	 * @Given /^user "([^"]*)" has changed the quota of the "([^"]*)" space to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param int $newQuota
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userHasChangedTheQuotaOfTheSpaceTo(
		string $user,
		string $spaceName,
		int $newQuota
	): void {
		$this->updateSpaceQuota($user, $spaceName, $newQuota);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @When /^user "([^"]*)" sets the file "([^"]*)" as a (description|space image)\s? in a special section of the "([^"]*)" space$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $type
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function updateSpaceSpecialSection(
		string $user,
		string $file,
		string $type,
		string $spaceName
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$spaceId = $space["id"];
		$fileId = $this->getFileId($user, $spaceName, $file);

		if ($type === "description") {
			$type = "readme";
		} else {
			$type = "image";
		}

		$bodyData = ["special" => [["specialFolder" => ["name" => "$type"], "id" => "$fileId"]]];
		$body = json_encode($bodyData, JSON_THROW_ON_ERROR);

		$this->featureContext->setResponse(
			GraphHelper::updateSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$body,
				$spaceId,
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has set the file "([^"]*)" as a (description|space image)\s? in a special section of the "([^"]*)" space$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $type
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userHasUpdatedSpaceSpecialSection(
		string $user,
		string $file,
		string $type,
		string $spaceName
	): void {
		$this->updateSpaceSpecialSection($user, $file, $type, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a space "([^"]*)" of type "([^"]*)" with quota "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string|null $spaceType
	 * @param int|null $quota
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedSpace(
		string $user,
		string $spaceName,
		string $spaceType,
		int $quota
	): void {
		$space = ["Name" => $spaceName, "driveType" => $spaceType, "quota" => ["total" => $quota]];
		$this->featureContext->setResponse($this->createSpace($user, $space));
		$this->setSpaceCreator($spaceName, $user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201 (Created)"
		);
	}

	/**
	 * @Given /^user "([^"]*)" has created a space "([^"]*)" with the default quota using the Graph API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function theUserHasCreatedASpaceByDefaultUsingTheGraphApi(
		string $user,
		string $spaceName
	): void {
		$space = ["Name" => $spaceName];
		$this->featureContext->setResponse($this->createSpace($user, $space));
		$this->setSpaceCreator($spaceName, $user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			"Expected response status code should be 201 (Created)"
		);
	}

	/**
	 * @param string $user
	 * @param string $space
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function createSpace(
		string $user,
		array $space
	): ResponseInterface {
		$body = json_encode($space, JSON_THROW_ON_ERROR);
		return GraphHelper::createSpace(
			$this->featureContext->getBaseUrl(),
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCopiesFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	):void {
		$space = $this->getSpaceByName($user, $spaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$spaceName
		);

		$fullUrl = $space["root"]["webDavUrl"] . '/' . ltrim($fileSource, "/");
		$this->featureContext->setResponse($this->copyFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @When /^user "([^"]*)" moves (?:file|folder) "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userMovesFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	):void {
		$space = $this->getSpaceByName($user, $spaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName(
			$user,
			$fileDestination,
			$spaceName
		);
		$headers['Overwrite'] = 'F';

		$fileSource = $this->escapePath(\trim($fileSource, "/"));
		$fullUrl = $space["root"]["webDavUrl"] . '/' . $fileSource;
		$this->featureContext->setResponse($this->moveFilesAndFoldersRequest($user, $fullUrl, $headers));
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Given /^user "([^"]*)" has moved (?:file|folder) "([^"]*)" to "([^"]*)" in space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fileDestination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasMovedFileWithinSpaceUsingTheWebDAVAPI(
		string $user,
		string $fileSource,
		string $fileDestination,
		string $spaceName
	):void {
		$this->userMovesFileWithinSpaceUsingTheWebDAVAPI(
			$user,
			$fileSource,
			$fileDestination,
			$spaceName
		);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			201,
			__METHOD__ . "Expected response status code should be 201 (Created)\n" .
			"Actual response status code was: " . $this->featureContext->getResponse()->getStatusCode() . "\n" .
			"Actual response body was: " . $this->featureContext->getResponse()->getBody()
		);
	}

	/**
	 * MOVE request for files|folders
	 *
	 * @param string $user
	 * @param string $fullUrl
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function moveFilesAndFoldersRequest(string $user, string $fullUrl, array $headers): ResponseInterface {
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'MOVE',
			$user,
			$this->featureContext->getPasswordForUser($user),
			$headers,
		);
	}

	/**
	 * @When /^user "([^"]*)" copies (?:file|folder) "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userCopiesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName
	):void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName($user, $fileDestination, $toSpaceName);
		$fullUrl = $space["root"]["webDavUrl"] . '/' . ltrim($fileSource, "/");
		$this->featureContext->setResponse($this->copyFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @When /^user "([^"]*)" overwrites file "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" while (copying|moving)\s? using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 * @param string $action
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userOverwritesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName,
		string $action
	):void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName($user, $fileDestination, $toSpaceName);
		$headers['Overwrite'] = 'T';
		$fullUrl = $space["root"]["webDavUrl"] . '/' . ltrim($fileSource, "/");
		if ($action === 'copying') {
			$response = $this->copyFilesAndFoldersRequest($user, $fullUrl, $headers);
		} else {
			$response = $this->moveFilesAndFoldersRequest($user, $fullUrl, $headers);
		}
		$this->featureContext->setResponse($response);
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * @Then /^user "([^"]*)" should not be able to download file "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileName
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userShouldNotBeAbleToDownloadFileInsideSpace(
		string $user,
		string $fileName,
		string $spaceName
	):void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->downloadFileAsUserUsingPassword($user, $fileName, $this->featureContext->getPasswordForUser($user));
		Assert::assertGreaterThanOrEqual(
			400,
			$this->featureContext->getResponse()->getStatusCode(),
			__METHOD__
			. ' download must fail'
		);
		Assert::assertLessThanOrEqual(
			499,
			$this->featureContext->getResponse()->getStatusCode(),
			__METHOD__
			. ' 4xx error expected but got status code "'
			. $this->featureContext->getResponse()->getStatusCode() . '"'
		);
	}

	/**
	 * @When /^user "([^"]*)" moves (?:file|folder) "([^"]*)" from space "([^"]*)" to "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $fileSource
	 * @param string $fromSpaceName
	 * @param string $fileDestination
	 * @param string $toSpaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userMovesFileFromAndToSpaceBetweenSpaces(
		string $user,
		string $fileSource,
		string $fromSpaceName,
		string $fileDestination,
		string $toSpaceName
	):void {
		$space = $this->getSpaceByName($user, $fromSpaceName);
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName($user, $fileDestination, $toSpaceName);
		$fullUrl = $space["root"]["webDavUrl"] . '/' . \ltrim($fileSource, "/");
		$this->featureContext->setResponse($this->moveFilesAndFoldersRequest($user, $fullUrl, $headers));
		$this->featureContext->pushToLastHttpStatusCodesArray();
	}

	/**
	 * returns a URL for destination with spacename
	 *
	 * @param string $user
	 * @param string $fileDestination
	 * @param string $spaceName
	 * @param string|null $endPath
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function destinationHeaderValueWithSpaceName(string $user, string $fileDestination, string $spaceName, string $endPath = null):string {
		$space = $this->getSpaceByName($user, $spaceName);
		$fileDestination = $this->escapePath(\ltrim($fileDestination, "/"));
		if ($endPath && str_contains($endPath, 'remote.php')) {
			// this is check for when we want to test with the endpoint having `remote.php` in  space webdav
			// by default spaces webdav is '/dav/spaces'
			return $this->featureContext->getBaseUrl() . '/remote.php/dav/spaces/' . $space['id'] . '/' . $fileDestination;
		}
		return $space["root"]["webDavUrl"] . '/' . $fileDestination;
	}

	/**
	 * COPY request for files|folders
	 *
	 * @param string $user
	 * @param string $fullUrl
	 * @param array $headers
	 *
	 * @return ResponseInterface
	 * @throws GuzzleException
	 */
	public function copyFilesAndFoldersRequest(string $user, string $fullUrl, array $headers): ResponseInterface {
		return HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			'COPY',
			$user,
			$this->featureContext->getPasswordForUser($user),
			$headers,
		);
	}

	/**
	 * @When /^user "([^"]*)" copies a file "([^"]*)" into "([^"]*)" inside space "([^"]*)" using file-id path "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $sourceFile
	 * @param string $destinationFile
	 * @param string $toSpaceName
	 * @param string $url
	 *
	 * @throws GuzzleException
	 * @return void
	 */
	public function userCopiesFileWithFileIdFromAndToSpaceBetweenSpaces(string $user, string $sourceFile, string $destinationFile, string $toSpaceName, string $url): void {
		// split the source when there are sub-folders
		$sourceFile = explode("/", $sourceFile);
		$fileDestination = $this->escapePath(\ltrim($destinationFile, "/")) . '/' . $this->escapePath(\ltrim(end($sourceFile), "/"));
		$headers['Destination'] = $this->destinationHeaderValueWithSpaceName($user, $fileDestination, $toSpaceName, $url);
		$fullUrl = $this->featureContext->getBaseUrl() . $url;
		$this->featureContext->setResponse($this->copyFilesAndFoldersRequest($user, $fullUrl, $headers));
	}

	/**
	 * @Given /^user "([^"]*)" has uploaded a file inside space "([^"]*)" with content "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $fileContent
	 * @param string $destination
	 *
	 * @return string[]
	 * @throws GuzzleException
	 */
	public function userHasUploadedFile(
		string $user,
		string $spaceName,
		string $fileContent,
		string $destination
	): array {
		$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		$fileId = $this->theUserUploadsAFileToSpace($user, $spaceName, $fileContent, $destination);
		$this->featureContext->theHTTPStatusCodeShouldBeOr(201, 204);
		return $fileId;
	}

	/**
	 * @When /^user "([^"]*)" shares a space "([^"]*)" with settings:$/
	 * @When /^user "([^"]*)" updates the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendShareSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$this->featureContext->setResponse($this->shareSpace($user, $spaceName, $rows));
	}

	/**
	 * @param string $user
	 * @param string $spaceName
	 * @param array $rows
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 */
	public function shareSpace(string $user, string $spaceName, array $rows): ResponseInterface {
		$space = $this->getSpaceByName($user, $spaceName);
		$availableRoleToAssignToShareSpace = ['manager', 'editor', 'viewer'];
		if (isset($rows['role']) && !\in_array(\strtolower($rows['role']), $availableRoleToAssignToShareSpace)) {
			throw new Error("The Selected " . $rows['role'] . " Cannot be Found");
		}
		$rows["role"] = \array_key_exists("role", $rows) ? $rows["role"] : "viewer";
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 7;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space["id"],
			"shareType" => $rows["shareType"],
			"shareWith" => $rows["shareWith"],
			"role" => $rows["role"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->baseUrl . $this->ocsApiUrl;

		return $this->sendPostRequestToUrl(
			$fullUrl,
			$user,
			$this->featureContext->getPasswordForUser($user),
			$body,
			$this->featureContext->getStepLineRef()
		);
	}

	/**
	 * @When /^user "([^"]*)" expires the (user|group) share of space "([^"]*)" for (?:user|group) "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $shareType
	 * @param  string $spaceName
	 * @param  string $memberUser
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userExpiresTheShareOfSpaceForUser(string $user, string $shareType, string $spaceName, string $memberUser) {
		$dateTime = new DateTime('yesterday');
		$rows['expireDate'] = $dateTime->format('Y-m-d\\TH:i:sP');
		$rows['shareWith'] = $memberUser;
		$rows['shareType'] = ($shareType === 'user') ? 7 : 8;
		$this->featureContext->setResponse($this->shareSpace($user, $spaceName, $rows));
	}

	/**
	 * @When /^user "([^"]*)" creates a share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function createShareResource(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$rows["path"] = \array_key_exists("path", $rows) ? $rows["path"] : null;
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 0;
		$rows["role"] = \array_key_exists("role", $rows) ? $rows["role"] : 'viewer';
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $this->getResourceId($user, $spaceName, $rows["path"]),
			"shareWith" => $rows["shareWith"],
			"shareType" => $rows["shareType"],
			"expireDate" => $rows["expireDate"],
			"role" => $rows["role"]
		];

		$fullUrl = $this->baseUrl . $this->ocsApiUrl;
		$this->featureContext->setResponse(
			$this->sendPostRequestToUrl(
				$fullUrl,
				$user,
				$this->featureContext->getPasswordForUser($user),
				$body,
				$this->featureContext->getStepLineRef()
			)
		);
		$response = $this->featureContext->getResponseXml(null, __METHOD__);
		$this->featureContext->addToCreatedPublicShares($response->data);
	}

	/**
	 * @Given /^user "([^"]*)" has created a share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function hasSharedTheFollowingEntityInsideOfSpace(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$this->createShareResource($user, $spaceName, $table);
		Assert::assertEquals(
			$this->featureContext->getResponse()->getStatusCode(),
			200,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @When /^user "([^"]*)" changes the last share with settings:$/
	 *
	 * @param  string $user
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function changeShareResourceWithSettings(
		string $user,
		TableNode $table
	): void {
		$rows = $table->getRowsHash();
		$this->featureContext->setResponse($this->updateSharedResource($user, $rows));
	}

	/**
	 * @param string $user
	 * @param array $rows
	 *
	 * @return ResponseInterface
	 *
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function updateSharedResource(string $user, array $rows):ResponseInterface {
		$shareId = (string) $this->featureContext->getLastCreatedPublicShare()->id;
		$fullUrl = $this->baseUrl . $this->ocsApiUrl . '/' . $shareId;
		return  HttpRequestHelper::sendRequest(
			$fullUrl,
			$this->featureContext->getStepLineRef(),
			"PUT",
			$this->featureContext->getActualUsername($user),
			$this->featureContext->getPasswordForUser($user),
			null,
			$rows
		);
	}

	/**
	 * @When /^user "([^"]*)" expires the last share$/
	 *
	 * @param string $user
	 *
	 * @return void
	 */
	public function userExpiresLastResourceShare(string $user): void {
		$dateTime = new DateTime('yesterday');
		$rows['expireDate'] = $dateTime->format('Y-m-d\\TH:i:sP');
		$this->featureContext->setResponse($this->updateSharedResource($user, $rows));
	}
	/**
	 * @When /^user "([^"]*)" creates a public link share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function createPublicLinkToEntityInsideOfSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$rows = $table->getRowsHash();

		$rows["path"] = \array_key_exists("path", $rows) ? $rows["path"] : null;
		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 3;
		$rows["permissions"] = \array_key_exists("permissions", $rows) ? $rows["permissions"] : null;
		$rows["password"] = \array_key_exists("password", $rows) ? $rows["password"] : null;
		$rows["name"] = \array_key_exists("name", $rows) ? $rows["name"] : null;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space['id'] . "/" . $rows["path"],
			"shareType" => $rows["shareType"],
			"permissions" => $rows["permissions"],
			"password" => $rows["password"],
			"name" => $rows["name"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->baseUrl . $this->ocsApiUrl;

		$this->featureContext->setResponse(
			$this->sendPostRequestToUrl(
				$fullUrl,
				$user,
				$this->featureContext->getPasswordForUser($user),
				$body,
				$this->featureContext->getStepLineRef()
			)
		);

		$response = $this->featureContext->getResponseXml(null, __METHOD__);
		$this->featureContext->addToCreatedPublicShares($response->data);
	}

	/**
	 * @Given /^user "([^"]*)" has created a public link share inside of space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedPublicLinkToEntityInsideOfSpaceRequest(
		string $user,
		string $spaceName,
		TableNode $table
	): void {
		$this->createPublicLinkToEntityInsideOfSpaceRequest($user, $spaceName, $table);

		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
	}

	/**
	 * @Given /^user "([^"]*)" has shared a space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  TableNode $tableNode
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasSharedSpace(
		string $user,
		string $spaceName,
		TableNode $tableNode
	): void {
		$this->sendShareSpaceRequest($user, $spaceName, $tableNode);
		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
		$expectedOCSStatus = "200";
		$this->ocsContext->theOCSStatusCodeShouldBe($expectedOCSStatus, "Expected OCS response status code $expectedOCSStatus");
	}

	/**
	 * @Given user :user has unshared a space :spaceName shared with :recipient
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $recipient
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasUnsharedASpaceSharedWith(
		string $user,
		string $spaceName,
		string $recipient
	): void {
		$this->sendUnshareSpaceRequest($user, $spaceName, $recipient);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @When /^user "([^"]*)" unshares a space "([^"]*)" to (?:user|group) "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $recipient
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendUnshareSpaceRequest(
		string $user,
		string $spaceName,
		string $recipient
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$fullUrl = $this->baseUrl . $this->ocsApiUrl . "/" . $space['id'] . "?shareWith=" . $recipient;

		$this->featureContext->setResponse(
			HttpRequestHelper::delete(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user)
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" removes the (?:file|folder) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendRemoveObjectFromSpaceRequest(
		string $user,
		string $object,
		string $spaceName
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$spaceWebDavUrl = $space["root"]["webDavUrl"] . '/' . ltrim($object, "/");
		$this->featureContext->setResponse(
			HttpRequestHelper::delete(
				$spaceWebDavUrl,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user)
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has deleted a space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasDeletedASpaceOwnedByUser(
		string $user,
		string $spaceName
	):void {
		$this->sendDeleteSpaceRequest($user, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			204,
			"Expected response status code should be 200"
		);
	}

	/**
	 * @When /^user "([^"]*)" disables a space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:disables|tries to disable) a space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendDisableSpaceRequest(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
		$this->featureContext->setResponse(
			GraphHelper::disableSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$space["id"],
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has removed the (?:file|folder) "([^"]*)" from space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendUserHasRemovedObjectFromSpaceRequest(
		string $user,
		string $object,
		string $spaceName
	): void {
		$this->sendRemoveObjectFromSpaceRequest($user, $object, $spaceName);
		$expectedHTTPStatus = "204";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
	}

	/**
	 * @Given /^user "([^"]*)" has disabled a space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendUserHasDisabledSpaceRequest(
		string $user,
		string $spaceName
	): void {
		$this->sendDisableSpaceRequest($user, $spaceName);
		$expectedHTTPStatus = "204";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
	}

	/**
	 * @When /^user "([^"]*)" deletes a space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:deletes|tries to delete) a space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendDeleteSpaceRequest(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);

		$this->featureContext->setResponse(
			GraphHelper::deleteSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$space["id"],
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" restores a disabled space "([^"]*)"$/
	 * @When /^user "([^"]*)" (?:restores|tries to restore) a disabled space "([^"]*)" owned by user "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param  string $owner
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendRestoreSpaceRequest(
		string $user,
		string $spaceName,
		string $owner = ''
	): void {
		$space = $this->getSpaceByName(($owner !== "") ? $owner : $user, $spaceName);
		$this->featureContext->setResponse(
			GraphHelper::restoreSpace(
				$this->featureContext->getBaseUrl(),
				$user,
				$this->featureContext->getPasswordForUser($user),
				$space["id"],
				$this->featureContext->getStepLineRef()
			)
		);
	}

	/**
	 * @Given /^user "([^"]*)" has restored a disabled space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasRestoredSpaceRequest(
		string $user,
		string $spaceName
	): void {
		$this->sendRestoreSpaceRequest($user, $spaceName);
		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
	}

	/**
	 * @When /^user "([^"]*)" lists all deleted files in the trash bin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userListAllDeletedFilesInTrash(
		string $user,
		string $spaceName
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$fullUrl = $this->baseUrl . $this->davSpacesUrl . "trash-bin/" . $space["id"];
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest($fullUrl, $this->featureContext->getStepLineRef(), 'PROPFIND', $user, $this->featureContext->getPasswordForUser($user))
		);
	}

	/**
	 * @When /^user "([^"]*)" with admin permission lists all deleted files in the trash bin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function adminListAllDeletedFilesInTrash(
		string $user,
		string $spaceName
	): void {
		$space = $this->getSpaceByNameManager($user, $spaceName);
		$fullUrl = $this->baseUrl . $this->davSpacesUrl . "trash-bin/" . $space["id"];
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest($fullUrl, $this->featureContext->getStepLineRef(), 'PROPFIND', $user, $this->featureContext->getPasswordForUser($user))
		);
	}

	/**
	 * User get all objects in the trash of project space
	 *
	 * Method "getTrashbinContentFromResponseXml" borrowed from core repository
	 * and return array like:
	 * 	[1] => Array
	 *       (
	 *             [href] => /remote.php/dav/spaces/trash-bin/spaceId/objectId/
	 *             [name] => deleted folder
	 *             [mtime] => 1649147272
	 *             [original-location] => deleted folder
	 *        )
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 *
	 * @return array
	 * @throws GuzzleException
	 */
	public function getObjectsInTrashbin(
		string $user,
		string $spaceName
	): array {
		$this->userListAllDeletedFilesInTrash($user, $spaceName);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			207,
			"Expected response status code should be 207"
		);
		return $this->trashbinContext->getTrashbinContentFromResponseXml(
			$this->featureContext->getResponseXml($this->featureContext->getResponse())
		);
	}

	/**
	 * @Then /^as "([^"]*)" (?:file|folder|entry) "([^"]*)" should (not|)\s?exist in the trashbin of the space "([^"]*)"$/
	 *
	 * @param  string $user
	 * @param  string $object
	 * @param  string $shouldOrNot   (not|)
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function checkExistenceOfObjectsInTrashbin(
		string $user,
		string $object,
		string $shouldOrNot,
		string $spaceName
	): void {
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);

		$expectedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$expectedObject = $objectInTrash["name"];
			}
		}
		if ($shouldOrNot === "not") {
			Assert::assertEmpty($expectedObject, "$object is found in the trash, but should not be there");
		} else {
			Assert::assertNotEmpty($expectedObject, "$object is not found in the trash");
		}
	}

	/**
	 * @When /^user "([^"]*)" restores the (?:file|folder) "([^"]*)" from the trash of the space "([^"]*)" to "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $object
	 * @param string $spaceName
	 * @param string $destination
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userRestoresSpaceObjectsFromTrashRequest(
		string $user,
		string $object,
		string $spaceName,
		string $destination
	): void {
		$space = $this->getSpaceByName($user, $spaceName);

		// find object in trash
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);
		$pathToDeletedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$pathToDeletedObject = $objectInTrash["href"];
			}
		}

		if ($pathToDeletedObject === "") {
			throw new Exception(__METHOD__ . " Object '$object' was not found in the trashbin of space '$spaceName' by user '$user'");
		}

		$destination = $this->baseUrl . $this->davSpacesUrl . $space["id"] . $destination;
		$header = ["Destination" => $destination, "Overwrite" => "F"];

		$fullUrl = $this->baseUrl . $pathToDeletedObject;
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				'MOVE',
				$user,
				$this->featureContext->getPasswordForUser($user),
				$header,
				""
			)
		);
	}

	/**
	 * @When user :user deletes the file/folder :resource from the trash of the space :spaceName
	 * @When user :user tries to delete the file/folder :resource from the trash of the space :spaceName
	 *
	 * @param string $user
	 * @param string $object
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws Exception
	 */
	public function userDeletesObjectsFromTrashRequest(
		string $user,
		string $object,
		string $spaceName
	): void {
		// find object in trash
		$objectsInTrash = $this->getObjectsInTrashbin($user, $spaceName);
		$pathToDeletedObject = "";
		foreach ($objectsInTrash as $objectInTrash) {
			if ($objectInTrash["name"] === $object) {
				$pathToDeletedObject = $objectInTrash["href"];
			}
		}

		if ($pathToDeletedObject === "") {
			throw new Exception(__METHOD__ . " Object '$object' was not found in the trashbin of space '$spaceName' by user '$user'");
		}

		$fullUrl = $this->baseUrl . $pathToDeletedObject;
		$this->featureContext->setResponse(
			HttpRequestHelper::sendRequest(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				'DELETE',
				$user,
				$this->featureContext->getPasswordForUser($user),
				[],
				""
			)
		);
	}

	/**
	 * User downloads a preview of the file inside the project space
	 *
	 * @When /^user "([^"]*)" downloads the preview of "([^"]*)" of the space "([^"]*)" with width "([^"]*)" and height "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $spaceName
	 * @param  string $width
	 * @param  string $height
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadPreview(
		string $user,
		string $fileName,
		string $spaceName,
		string $width,
		string $height
	): void {
		$eTag = str_replace("\"", "", $this->getETag($user, $spaceName, $fileName));
		$urlParameters = [
			'scalingup' => 0,
			'preview' => '1',
			'a' => '1',
			'c' => $eTag,
			'x' => $width,
			'y' => $height
		];
		$urlParameters = \http_build_query($urlParameters, '', '&');
		$space = $this->getSpaceByName($user, $spaceName);

		$fullUrl = $this->baseUrl . $this->davSpacesUrl . $space['id'] . '/' . $fileName . '?' . $urlParameters;

		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$fullUrl,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user)
			)
		);
	}

	/**
	 * @When /^user "([^"]*)" downloads the file "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadFile(
		string $user,
		string $fileName,
		string $spaceName
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->downloadFileAsUserUsingPassword($user, $fileName, $this->featureContext->getPasswordForUser($user));
	}

	/**
	 * @When /^user "([^"]*)" requests the checksum of (?:file|folder|entry) "([^"]*)" in space "([^"]*)" via propfind using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userRequestsTheChecksumViaPropfindInSpace(
		string $user,
		string $path,
		string $spaceName
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->checksumContext->userRequestsTheChecksumOfViaPropfind($user, $path);
	}

	/**
	 * @When /^user "([^"]*)" uploads file with checksum "([^"]*)" and content "([^"]*)" to "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $checksum
	 * @param string $content
	 * @param string $destination
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userUploadsFileWithChecksumWithContentInSpace(
		string $user,
		string $checksum,
		string $content,
		string $destination,
		string $spaceName
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->userUploadsAFileWithChecksumAndContentTo($user, $checksum, $content, $destination);
	}

	/**
	 * @When /^user "([^"]*)" downloads version of the file "([^"]*)" with the index "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param  string $user
	 * @param  string $fileName
	 * @param  string $index
	 * @param  string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function downloadVersionOfTheFile(
		string $user,
		string $fileName,
		string $index,
		string $spaceName
	): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->setResponse($this->filesVersionsContext->downloadVersion($user, $fileName, $index));
		WebDavHelper::$SPACE_ID_FROM_OCIS = '';
	}

	/**
	 * @When /^user "([^"]*)" tries to get version of the file "([^"]*)" with the index "([^"]*)" of the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $file
	 * @param string $index
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws Exception
	 */
	public function userTriesToDownloadFileVersions(string $user, string $file, string $index, string $spaceName):void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->filesVersionsContext->userGetsFileVersions($user, $file);
	}

	/**
	 * return the etag for an element inside a space
	 *
	 * @param string $user requestor
	 * @param string $space space name
	 * @param string $path path to the file
	 *
	 * @return string
	 * @throws GuzzleException
	 */
	public function userGetsEtagOfElementInASpace(string $user, string $space, string $path) {
		$this->setSpaceIDByName($user, $space);
		$this->webDavPropertiesContext->storeEtagOfElement($user, $path);
		return $this->featureContext->getEtagFromResponseXmlObject();
	}

	/**
	 * saves the etag of an element in a space
	 *
	 * @param string $user requestor
	 * @param string $space space name
	 * @param string $path path to the file
	 * @param ?string $storePath path to the file in the store
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function storeEtagOfElementInSpaceForUser(string $user, string $space, string $path, ?string $storePath = ""): void {
		if ($storePath === "") {
			$storePath = $path;
		}
		$this->storedEtags[$user][$space][$storePath]
			= $this->userGetsEtagOfElementInASpace(
				$user,
				$space,
				$path,
			);
	}

	/**
	 * returns stored etag for an element if present
	 *
	 * @param string $user
	 * @param string $space
	 * @param string $path
	 *
	 * @return string
	 */
	public function getStoredEtagForPathInSpaceOfAUser(string $user, string $space, string $path): string {
		Assert::assertArrayHasKey(
			$user,
			$this->storedEtags,
			__METHOD__ . " No stored etags for user '$user' found"
			. "\nFound: " . print_r($this->storedEtags, true)
		);
		Assert::assertArrayHasKey(
			$space,
			$this->storedEtags[$user],
			__METHOD__ . " No stored etags for user '$user' with space '$space' found"
			. "\nFound: " . implode(', ', array_keys($this->storedEtags[$user]))
		);
		Assert::assertArrayHasKey(
			$path,
			$this->storedEtags[$user][$space],
			__METHOD__ . " No stored etags for user '$user' with space '$space' with path '$path' found"
			. '\nFound: ' . print_r($this->storedEtags[$user][$space], true)
		);
		return $this->storedEtags[$user][$space][$path];
	}

	/**
	 * @Then /^these etags (should|should not) have changed$/
	 *
	 * @param string $action
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function theseEtagsShouldShouldNotHaveChanged(string $action, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ["user", "path", "space"]);
		$this->featureContext->verifyTableNodeColumnsCount($table, 3);
		$changedEtagCount = 0;
		$changedEtagMessage = __METHOD__;
		foreach ($table->getColumnsHash() as $row) {
			$user = $row['user'];
			$path = $row['path'];
			$space = $row['space'];
			$etag = $this->userGetsEtagOfElementInASpace($user, $space, $path);
			$storedEtag = $this->getStoredEtagForPathInSpaceOfAUser($user, $space, $path);
			if ($action === 'should' && $etag === $storedEtag) {
				$changedEtagCount++;
				$changedEtagMessage .= "\nExpected etag of element '$path' for  user '$user' in space '$space' to change, but it did not.";
			}
			if ($action === 'should not' && $etag !== $storedEtag) {
				$changedEtagCount++;
				$changedEtagMessage .= "\nExpected etag of element '$path' for  user '$user' in space '$space' to change, but it did not.";
			}
		}

		Assert::assertEquals(0, $changedEtagCount, $changedEtagMessage);
	}

	/**
	 * @Given /^user "([^"]*)" has stored etag of element "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $space
	 *
	 * @return void
	 * @throws GuzzleException | Exception
	 */
	public function userHasStoredEtagOfElementFromSpace(string $user, string $path, string $space):void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElementInSpaceForUser(
			$user,
			$space,
			$path,
		);
		if ($this->storedEtags[$user][$space][$path] === "" || $this->storedEtags[$user][$space][$path] === null) {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 * @Given /^user "([^"]*)" has stored etag of element "([^"]*)" on path "([^"]*)" inside space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $storePath
	 * @param string $space
	 *
	 * @return void
	 * @throws Exception | GuzzleException
	 */
	public function userHasStoredEtagOfElementOnPathFromSpace(string $user, string $path, string $storePath, string $space):void {
		$user = $this->featureContext->getActualUsername($user);
		$this->storeEtagOfElementInSpaceForUser(
			$user,
			$space,
			$path,
			$storePath
		);
		if ($this->storedEtags[$user][$space][$storePath] === "" || $this->storedEtags[$user][$space][$storePath] === null) {
			throw new Exception("Expected stored etag to be some string but found null!");
		}
	}

	/**
	 * @When /^user "([^"]*)" creates a public link share of the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode|null $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function sendShareSpaceViaLinkRequest(
		string $user,
		string $spaceName,
		?TableNode $table
	): void {
		$space = $this->getSpaceByName($user, $spaceName);
		$rows = $table->getRowsHash();

		$rows["shareType"] = \array_key_exists("shareType", $rows) ? $rows["shareType"] : 3;
		$rows["permissions"] = \array_key_exists("permissions", $rows) ? $rows["permissions"] : null;
		$rows["password"] = \array_key_exists("password", $rows) ? $rows["password"] : null;
		$rows["name"] = \array_key_exists("name", $rows) ? $rows["name"] : null;
		$rows["expireDate"] = \array_key_exists("expireDate", $rows) ? $rows["expireDate"] : null;

		$body = [
			"space_ref" => $space['id'],
			"shareType" => $rows["shareType"],
			"permissions" => $rows["permissions"],
			"password" => $rows["password"],
			"name" => $rows["name"],
			"expireDate" => $rows["expireDate"]
		];

		$fullUrl = $this->baseUrl . $this->ocsApiUrl;

		$this->featureContext->setResponse(
			$this->sendPostRequestToUrl(
				$fullUrl,
				$user,
				$this->featureContext->getPasswordForUser($user),
				$body,
				$this->featureContext->getStepLineRef()
			)
		);

		$response = $this->featureContext->getResponseXml(null, __METHOD__);
		$this->featureContext->addToCreatedPublicShares($response->data);
	}

	/**
	 * @Given /^user "([^"]*)" has created a public link share of the space "([^"]*)" with settings:$/
	 *
	 * @param  string $user
	 * @param  string $spaceName
	 * @param TableNode|null $table
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userHasCreatedPublicLinkShareOfSpace(
		string $user,
		string $spaceName,
		?TableNode $table
	): void {
		$this->sendShareSpaceViaLinkRequest($user, $spaceName, $table);

		$expectedHTTPStatus = "200";
		$this->featureContext->theHTTPStatusCodeShouldBe(
			$expectedHTTPStatus,
			"Expected response status code should be $expectedHTTPStatus"
		);
	}

	/**
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain the last created public link$/
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain the last created public link of the file "([^"]*)"$/
	 * @Then /^for user "([^"]*)" the space "([^"]*)" should (not|)\s?contain the last created share of the file "([^"]*)"$/
	 *
	 * @param string    $user
	 * @param string    $spaceName
	 * @param string    $shouldOrNot   (not|)
	 * @param string    $fileName
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function forUserSpaceShouldContainLinks(
		string $user,
		string $spaceName,
		string $shouldOrNot,
		string $fileName = ''
	): void {
		$body = '';
		if (!empty($fileName)) {
			$body = $this->getFileId($user, $spaceName, $fileName);
		} else {
			$space = $this->getSpaceByName($user, $spaceName);
			$body = $space['id'];
		}

		$url = "/apps/files_sharing/api/v1/shares?reshares=true&space_ref=" . $body;

		$this->ocsContext->userSendsHTTPMethodToOcsApiEndpointWithBody(
			$user,
			'GET',
			$url,
		);

		$should = ($shouldOrNot !== "not");
		$responseArray = json_decode(json_encode($this->featureContext->getResponseXml()->data), true, 512, JSON_THROW_ON_ERROR);

		if ($should) {
			Assert::assertNotEmpty($responseArray, __METHOD__ . ' Response should contain a link, but it is empty');
			foreach ($responseArray as $element) {
				$expectedLinkId = (string) $this->featureContext->getLastCreatedPublicShare()->id;
				Assert::assertEquals($element["id"], $expectedLinkId, "link IDs are different");
			}
		} else {
			Assert::assertEmpty($responseArray, __METHOD__ . ' Response should be empty');
		}
	}

	/**
	 * @When /^user "([^"]*)" gets the following properties of (?:file|folder|entry|resource) "([^"]*)" inside space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string    $user
	 * @param string    $resourceName
	 * @param string    $spaceName
	 * @param TableNode|null $propertiesTable
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userGetsTheFollowingPropertiesOfFileInsideSpaceUsingTheWebdavApi(
		string $user,
		string $resourceName,
		string $spaceName,
		TableNode $propertiesTable
	):void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->webDavPropertiesContext->userGetsPropertiesOfFolder($user, $resourceName, $propertiesTable);
	}

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry|resource) "([^"]*)" inside space "([^"]*)" should contain a property "([^"]*)" with value "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $resourceName
	 * @param string $spaceName
	 * @param string $property
	 * @param string $value
	 *
	 * @return void
	 *
	 * @throws Exception|GuzzleException
	 */
	public function userGetsTheFollowingPropertiesOfFileInsideSpaceWithValueUsingTheWebdavApi(
		string $user,
		string $resourceName,
		string $spaceName,
		string $property,
		string $value
	):void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->webDavPropertiesContext->asUserFolderShouldContainAPropertyWithValue($user, $resourceName, $property, $value);
	}

	/**
	 * @Then /^as user "([^"]*)" (?:file|folder|entry) "([^"]*)" inside space "([^"]*)" (should|should not) be favorited$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 * @param string $shouldOrNot
	 *
	 * @return void
	 */
	public function asUserFileOrFolderInsideSpaceShouldOrNotBeFavorited(string $user, string $path, string $spaceName, string $shouldOrNot):void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->favoritesContext->asUserFileOrFolderShouldBeFavorited($user, $path, ($shouldOrNot === 'should') ? 1 : 0);
	}

	/**
	 * @When /^user "([^"]*)" favorites element "([^"]*)" in space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function userFavoritesElementInSpaceUsingTheWebdavApi(string $user, string $path, string $spaceName): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->setResponse($this->favoritesContext->userFavoritesElement($user, $path));
	}

	/**
	 * @Given /^user "([^"]*)" has stored id of (file|folder) "([^"]*)" of the space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $fileOrFolder
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userHasStoredIdOfPathOfTheSpace(string $user, string $fileOrFolder, string $path, string $spaceName): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->userStoresFileIdForPath($user, $fileOrFolder, $path);
	}

	/**
	 * @Then /^user "([^"]*)" (folder|file) "([^"]*)" of the space "([^"]*)" should have the previously stored id$/
	 *
	 * @param string|null $user
	 * @param string $fileOrFolder
	 * @param string $path
	 * @param string $spaceName
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userFolderOfTheSpaceShouldHaveThePreviouslyStoredId(string $user, string $fileOrFolder, string $path, string $spaceName): void {
		$this->setSpaceIDByName($user, $spaceName);
		$this->featureContext->userFileShouldHaveStoredId($user, $fileOrFolder, $path);
	}

	/**
	 * @Then /^for user "([^"]*)" the search result should contain space "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 *
	 * @return void
	 * @throws GuzzleException
	 */
	public function searchResultShouldContainSpace(string $user, string $spaceName): void {
		// get a response after a Report request (called in the core)
		$responseArray = json_decode(json_encode($this->featureContext->getResponseXml()->xpath("//d:response/d:href")), true, 512, JSON_THROW_ON_ERROR);
		Assert::assertNotEmpty($responseArray, "search result is empty");

		// for mountpoint, id looks a little different than for project space
		if (str_contains($spaceName, 'mountpoint')) {
			$splitSpaceName = explode("/", $spaceName);
			$space = $this->getSpaceByName($user, $splitSpaceName[1]);
			$splitSpaceId = explode("$", $space['id']);
			$topWebDavPath = "/remote.php/dav/spaces/" . str_replace('!', '%21', $splitSpaceId[1]);
		} else {
			$space = $this->getSpaceByName($user, $spaceName);
			$topWebDavPath = "/remote.php/dav/spaces/" . $space['id'];
		}

		$spaceFound = false;
		foreach ($responseArray as $value) {
			if ($topWebDavPath === $value[0]) {
				$spaceFound = true;
			}
		}
		Assert::assertTrue($spaceFound, "response does not contain the space '$spaceName'");
	}

	/**
	 * @When /^user "([^"]*)" sends PROPFIND request to space "([^"]*)" using the WebDAV API$/
	 * @When /^user "([^"]*)" sends PROPFIND request from the space "([^"]*)" to the resource "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param ?string $resource
	 *
	 * @throws GuzzleException
	 *
	 * @return void
	 */
	public function userSendsPropfindRequestToSpace(string $user, string $spaceName, ?string $resource = ""): void {
		$this->setSpaceIDByName($user, $spaceName);
		$properties = ['oc:permissions','oc:file-parent','oc:fileid','oc:share-types','oc:privatelink','d:resourcetype','oc:size','oc:name','d:getcontenttype', 'oc:tags'];
		$this->featureContext->setResponse(
			WebDavHelper::propfind(
				$this->featureContext->getBaseUrl(),
				$this->featureContext->getActualUsername($user),
				$this->featureContext->getPasswordForUser($user),
				$resource,
				$properties,
				$this->featureContext->getStepLineRef(),
				"0",
				"files",
				WebDavHelper::DAV_VERSION_SPACES
			)
		);
	}

	/**
	 * @Then /^the "([^"]*)" response should contain a space "([^"]*)" with these key and value pairs:$/
	 *
	 * @param string $method # method should be either PROPFIND or REPORT
	 * @param string $space
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function theResponseShouldContainSpace(string $method, string $space, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ['key', 'value']);
		$this->theResponseShouldContain($method, $this->getSpaceCreator($space), $space, $table);
	}

	/**
	 * @Then /^the "([^"]*)" response to user "([^"]*)" should contain a mountpoint "([^"]*)" with these key and value pairs:$/
	 *
	 * @param string $method # method should be either PROPFIND or REPORT
	 * @param string $user
	 * @param string $mountPoint
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function theResponseShouldContainMountPoint(string $method, string $user, string $mountPoint, TableNode $table): void {
		$this->featureContext->verifyTableNodeColumns($table, ['key', 'value']);
		$this->theResponseShouldContain($method, $user, $mountPoint, $table);
	}

	/**
	 * @param string $method # method should be either PROPFIND or REPORT
	 * @param string $user
	 * @param string $spaceNameOrMountPoint # an entity inside a space, or the space name itself
	 * @param TableNode $table
	 *
	 * @return void
	 * @throws GuzzleException
	 * @throws JsonException
	 */
	public function theResponseShouldContain(string $method, string $user, string $spaceNameOrMountPoint, TableNode $table): void {
		$xmlRes = $this->featureContext->getResponseXml();
		foreach ($table->getHash() as $row) {
			$findItem = $row['key'];
			Assert::assertNotEmpty(
				$xmlRes->xpath("//d:response/d:propstat/d:prop/$findItem"),
				'The xml response "' . $xmlRes->asXML() . '" did not contain "<' . $findItem . '>" element'
			);
			$responseValue = $xmlRes->xpath("//d:response/d:propstat/d:prop/$findItem")[0]->__toString();
			$value = str_replace('UUIDof:', '', $row['value']);
			switch ($findItem) {
				case "oc:fileid":
					$resourceType = $xmlRes->xpath("//d:response/d:propstat/d:prop/d:getcontenttype")[0]->__toString();
					if ($method === 'PROPFIND') {
						if (!$resourceType) {
							Assert::assertEquals($this->getResourceId($user, $spaceNameOrMountPoint, $value), $responseValue, 'wrong fileId in the response');
						} else {
							Assert::assertEquals($this->getFileId($user, $spaceNameOrMountPoint, $value), $responseValue, 'wrong fileId in the response');
						}
					} else {
						if ($resourceType === 'httpd/unix-directory') {
							Assert::assertEquals($this->getResourceId($user, $spaceNameOrMountPoint, $value), $responseValue, 'wrong fileId in the response');
						} else {
							Assert::assertEquals($this->getFileId($user, $spaceNameOrMountPoint, $value), $responseValue, 'wrong fileId in the response');
						}
					}
					break;
				case "oc:file-parent":
					Assert::assertEquals($this->getResourceId($user, $spaceNameOrMountPoint, $value), $responseValue, 'wrong file-parentId in the response');
					break;
				case "oc:privatelink":
					Assert::assertEquals($this->getPrivateLink($user, $spaceNameOrMountPoint), $responseValue, 'cannot find private link for space or resource in the response');
					break;
				case "oc:tags":
					// The value should be a comma-separated string of tag names.
					// We do not care what order they happen to be in, so compare as sorted lists.
					$expectedTags = \explode(",", $value);
					$expectedTags = \sort($expectedTags);
					$actualTags = \explode(",", $responseValue);
					$actualTags = \sort($actualTags);
					Assert::assertEquals($expectedTags, $actualTags, "wrong $findItem in the response");
					break;
				default:
					Assert::assertEquals($value, $responseValue, "wrong $findItem in the response");
					break;
			}
		}
	}

	/**
	 * @When /^public downloads the folder "([^"]*)" from the last created public link using the public files API$/
	 *
	 * @param string $resource
	 *
	 * @return void
	 * @throws GuzzleException|JsonException
	 */
	public function publicDownloadsTheFolderFromTheLastCreatedPublicLink(string $resource) {
		$token = $this->featureContext->getLastCreatedPublicShareToken();
		$response = $this->featureContext->listFolderAndReturnResponseXml(
			$token,
			$resource,
			'0',
			['oc:fileid'],
			$this->featureContext->getDavPathVersion() === 1 ? "public-files" : "public-files-new"
		);
		$resourceId = json_decode(json_encode($response->xpath("//d:response/d:propstat/d:prop/oc:fileid")), true, 512, JSON_THROW_ON_ERROR);
		$queryString = 'public-token=' . $token . '&id=' . $resourceId[0][0];
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$this->featureContext->getBaseUrl() . '/archiver?' . $queryString,
				$this->featureContext->getStepLineRef(),
				'',
				'',
			)
		);
	}

	/**
	 * @Then the relative quota amount should be :quota_amount
	 *
	 * @param string $quotaAmount
	 *
	 * @return void
	 * @throws Exception
	 */
	public function theRelativeQuotaAmountShouldBe(string $quotaAmount): void {
		$data = $this->ocsContext->getOCSResponseData($this->featureContext->getResponse());
		if (isset($data->quota, $data->quota->relative)) {
			Assert::assertEquals(
				$data->quota->relative,
				$quotaAmount,
				"Expected relative quota amount to be $quotaAmount but found to be $data->quota->relative"
			);
		} else {
			throw new Exception(
				"No relative quota amount found in responseXml"
			);
		}
	}

	/**
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to (user|group)\s? "([^"]*)" with role "([^"]*)"$/
	 * @Then /^the user "([^"]*)" should have a space called "([^"]*)" granted to (user|group)\s? "([^"]*)" with role "([^"]*)" and expiration date "([^"]*)"$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $recipientType
	 * @param string $recipient
	 * @param string $role
	 * @param string|null $expirationDate
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function theUserShouldHaveSpaceWithRecipient(
		string $user,
		string $spaceName,
		string $recipientType,
		string $recipient,
		string $role,
		string $expirationDate = null
	): void {
		$this->theUserListsAllHisAvailableSpacesUsingTheGraphApi($user);
		$this->featureContext->theHTTPStatusCodeShouldBe(
			200,
			"Expected response status code should be 200"
		);
		Assert::assertIsArray($spaceAsArray = $this->getSpaceByNameFromResponse($spaceName), "No space with name $spaceName found");
		$recipientType === 'user' ? $recipientId = $this->featureContext->getUserIdByUserName($recipient) : $recipientId = $this->featureContext->getGroupIdByGroupName($recipient);
		$foundRoleInResponse = false;
		foreach ($spaceAsArray['root']['permissions'] as $permission) {
			if (isset($permission['grantedToIdentities'][0][$recipientType]) && $permission['roles'][0] === $role && $permission['grantedToIdentities'][0][$recipientType]['id'] === $recipientId) {
				$foundRoleInResponse = true;
				if ($expirationDate !== null && isset($permission['expirationDateTime'])) {
					Assert::assertEquals($expirationDate, (preg_split("/[\sT]+/", $permission['expirationDateTime']))[0], "$expirationDate is different in the response");
				}
				break;
			}
		}
		Assert::assertTrue($foundRoleInResponse, "the response does not contain the $recipientType $recipient");
	}

	/**
	 * @When user :user tries to download the space :spaceName owned by user :owner using the WebDAV API
	 * @When /^user "([^"]*)" (?:downloads|tries to download) the space "([^"]*)" using the WebDAV API$/
	 *
	 * @param string $user
	 * @param string $spaceName
	 * @param string $owner
	 *
	 * @return void
	 *
	 * @throws GuzzleException
	 */
	public function userDownloadsTheSpaceUsingTheWebdavApi(string $user, string $spaceName, string $owner = ''):void {
		$space = $this->getSpaceByName($owner ?: $user, $spaceName);
		$url = $this->featureContext->getBaseUrl() . '/archiver?id=' . $space['id'];
		$this->featureContext->setResponse(
			HttpRequestHelper::get(
				$url,
				$this->featureContext->getStepLineRef(),
				$user,
				$this->featureContext->getPasswordForUser($user),
			)
		);
	}
}
