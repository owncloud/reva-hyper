Feature: Remove files, folder
  As a user
  I want to be able to remove files, folders
  So that I can remove unnecessary objects

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "delete objects" with the default quota using the Graph API
    And user "Alice" has created a folder "folderForDeleting/sub1/sub2" in space "delete objects"
    And user "Alice" has uploaded a file inside space "delete objects" with content "some content" to "text.txt"


  Scenario Outline: user deletes a folder with some subfolders in a space via the webDav API
    Given user "Alice" has shared a space "delete objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    When user "<user>" removes the folder "folderForDeleting" from space "delete objects"
    Then the HTTP status code should be "<code>"
    And for user "<user>" the space "delete objects" <shouldOrNotBeInSpace> contain these entries:
      | folderForDeleting |
    And as "<user>" folder "folderForDeleting" <shouldOrNotBeInTrash> exist in the trashbin of the space "delete objects"
    Examples:
      | user  | role    | code | shouldOrNotBeInSpace | shouldOrNotBeInTrash |
      | Alice | manager | 204  | should not           | should               |
      | Brian | manager | 204  | should not           | should               |
      | Brian | editor  | 204  | should not           | should               |
      | Brian | viewer  | 403  | should               | should not           |


  Scenario Outline: user deletes a subfolder in a space via the webDav API
    Given user "Alice" has shared a space "delete objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    When user "<user>" removes the folder "folderForDeleting/sub1" from space "delete objects"
    Then the HTTP status code should be "<code>"
    And for user "<user>" the space "delete objects" should contain these entries:
      | folderForDeleting |
    And for user "<user>" folder "folderForDeleting/" of the space "delete objects" <shouldOrNotBeInSpace> contain these entries:
      | sub1 |
    And as "<user>" folder "sub1" <shouldOrNotBeInTrash> exist in the trashbin of the space "delete objects"
    Examples:
      | user  | role    | code | shouldOrNotBeInSpace | shouldOrNotBeInTrash |
      | Alice | manager | 204  | should not           | should               |
      | Brian | manager | 204  | should not           | should               |
      | Brian | editor  | 204  | should not           | should               |
      | Brian | viewer  | 403  | should               | should not           |


  Scenario Outline: user deletes a file in a space via the webDav API
    Given user "Alice" has shared a space "delete objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    When user "<user>" removes the file "text.txt" from space "delete objects"
    Then the HTTP status code should be "<code>"
    And for user "<user>" the space "delete objects" <shouldOrNotBeInSpace> contain these entries:
      | text.txt |
    And as "<user>" file "text.txt" <shouldOrNotBeInTrash> exist in the trashbin of the space "delete objects"
    Examples:
      | user  | role    | code | shouldOrNotBeInSpace | shouldOrNotBeInTrash |
      | Alice | manager | 204  | should not           | should               |
      | Brian | manager | 204  | should not           | should               |
      | Brian | editor  | 204  | should not           | should               |
      | Brian | viewer  | 403  | should               | should not           |


  Scenario: try to delete an empty string folder from a space
    When user "Alice" removes the folder "" from space "delete objects"
    Then the HTTP status code should be "405"
