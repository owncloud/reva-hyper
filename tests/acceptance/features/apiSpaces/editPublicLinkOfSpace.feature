Feature: A manager of the space can edit public link
  As an user with manager space role
  I want to be able to edit a public link.
  So that I can remove or change permission, password, expireDate, and name attributes
  Users without the manager role cannot see or edit the public link


  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "edit space" with the default quota using the Graph API
    And user "Alice" has created a public link share of the space "edit space" with settings:
      | permissions | 1                        |
      | password    | qwerty                   |
      | expireDate  | 2040-01-01T23:59:59+0100 |
      | name        | someName                 |
    And user "Alice" has uploaded a file inside space "edit space" with content "some content" to "test.txt"


  Scenario Outline: manager of the space can edit public link.
    Given using OCS API version "2"
    When user "Alice" updates the last public link share using the sharing API with
      | permissions | <permissions> |
      | password    | <password>    |
      | name        | <linkName>    |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Alice" should include
      | item_type         | folder                |
      | mimetype          | httpd/unix-directory  |
      | file_target       | /                     |
      | path              | /                     |
      | permissions       | <expectedPermissions> |
      | share_type        | public_link           |
      | displayname_owner | %displayname%         |
      | name              | <linkName>            |
    And the public should be able to download file "/test.txt" from inside the last public link shared folder using the new public WebDAV API with password "<password>"
    And the downloaded content should be "some content"
    Examples:
      | permissions | expectedPermissions       | password | linkName |
      | 5           | read,create               | newPass  |          |
      | 15          | read,update,create,delete |          | newName  |


  Scenario Outline: members can see a created public link
    Given using OCS API version "2"
    When user "Alice" shares a space "edit space" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And for user "Alice" the space "edit space" should contain the last created public link
    And for user "Brian" the space "edit space" should contain the last created public link
    Examples:
      | role    |
      | manager |
      | editor  |
      | viewer  |


  Scenario Outline: members of the space try to edit a public link
    Given using OCS API version "2"
    And user "Alice" has shared a space "edit space" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    When user "Brian" updates the last public link share using the sharing API with
      | permissions | 15 |
    Then the HTTP status code should be "<code>"
    And the OCS status code should be "<codeOCS>"
    Examples:
      | role    | code | codeOCS |
      | manager | 200  | 200     |
      | editor  | 401  | 997     |
      | viewer  | 401  | 997     |
