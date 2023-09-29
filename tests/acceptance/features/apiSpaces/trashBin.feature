Feature: Restore files, folder
  As a user with manager and editor role
  I want to be able to restore files, folders
  So that I can get the resources that were accidentally deleted

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "restore objects" with the default quota using the Graph API
    And user "Alice" has created a folder "newFolder" in space "restore objects"
    And user "Alice" has uploaded a file inside space "restore objects" with content "test" to "newFolder/file.txt"


  Scenario Outline: user with different role can see deleted objects in trash bin of the space via the webDav API
    Given user "Alice" has shared a space "restore objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    And user "Alice" has removed the file "newFolder/file.txt" from space "restore objects"
    And user "Alice" has removed the folder "newFolder" from space "restore objects"
    When user "Brian" lists all deleted files in the trash bin of the space "restore objects"
    Then the HTTP status code should be "207"
    And as "Brian" folder "newFolder" should exist in the trashbin of the space "restore objects"
    And as "Brian" file "file.txt" should exist in the trashbin of the space "restore objects"
    Examples:
      | role    |
      | manager |
      | editor  |
      | viewer  |


  Scenario Outline: user can restore a folder with some objects from the trash via the webDav API
    Given user "Alice" has shared a space "restore objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    And user "Alice" has removed the folder "newFolder" from space "restore objects"
    When user "<user>" restores the folder "newFolder" from the trash of the space "restore objects" to "/newFolder"
    Then the HTTP status code should be "<code>"
    And for user "<user>" the space "restore objects" <shouldOrNotBeInSpace> contain these entries:
      | newFolder |
    And as "<user>" folder "newFolder" <shouldOrNotBeInTrash> exist in the trashbin of the space "restore objects"
    Examples:
      | user  | role    | code | shouldOrNotBeInSpace | shouldOrNotBeInTrash |
      | Alice | manager | 201  | should               | should not           |
      | Brian | manager | 201  | should               | should not           |
      | Brian | editor  | 201  | should               | should not           |
      | Brian | viewer  | 403  | should not           | should               |


  Scenario Outline: user can restore a file from the trash via the webDav API
    Given user "Alice" has shared a space "restore objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    And user "Alice" has removed the file "newFolder/file.txt" from space "restore objects"
    When user "<user>" restores the file "file.txt" from the trash of the space "restore objects" to "newFolder/file.txt"
    Then the HTTP status code should be "<code>"
    And for user "<user>" folder "newFolder" of the space "restore objects" <shouldOrNotBeInSpace> contain these files:
      | file.txt |
    And as "<user>" file "file.txt" <shouldOrNotBeInTrash> exist in the trashbin of the space "restore objects"
    Examples:
      | user  | role    | code | shouldOrNotBeInSpace | shouldOrNotBeInTrash |
      | Alice | manager | 201  | should               | should not           |
      | Brian | manager | 201  | should               | should not           |
      | Brian | editor  | 201  | should               | should not           |
      | Brian | viewer  | 403  | should not           | should               |


  Scenario Outline: only space manager can purge the trash via the webDav API
    Given user "Alice" has shared a space "restore objects" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    And the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Alice" has removed the file "newFolder/file.txt" from space "restore objects"
    When user "Brian" deletes the file "file.txt" from the trash of the space "restore objects"
    Then the HTTP status code should be "<code>"
    And as "Brian" file "file.txt" <shouldOrNotBeInTrash> exist in the trashbin of the space "restore objects"
    Examples:
      | role    | code | shouldOrNotBeInTrash |
      | manager | 204  | should not           |
      | editor  | 403  | should               |
      | viewer  | 403  | should               |


  Scenario Outline: admin user who is not a member of space cannot see its trash bin
    Given user "Alice" has removed the file "newFolder/file.txt" from space "restore objects"
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When user "Brian" with admin permission lists all deleted files in the trash bin of the space "restore objects"
    Then the HTTP status code should be "404"
    Examples:
      | role        |
      | Space Admin |
      | Admin       |


  Scenario Outline: admin user without space-manager role cannot purge the trash
    Given user "Alice" has shared a space "restore objects" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    And user "Alice" has removed the file "newFolder/file.txt" from space "restore objects"
    When user "Brian" tries to delete the file "file.txt" from the trash of the space "restore objects"
    Then the HTTP status code should be "403"
    And as "Alice" file "file.txt" should exist in the trashbin of the space "restore objects"
    Examples:
      | role        |
      | Space Admin |
      | Admin       |

