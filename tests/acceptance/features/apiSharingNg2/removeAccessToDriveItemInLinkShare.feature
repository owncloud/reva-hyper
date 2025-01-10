Feature: Remove access to a drive item
  https://owncloud.dev/libre-graph-api/#/drives.permissions/DeletePermission

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario Outline: user removes access to a folder in link share
    Given user "Alice" has created folder "FolderToShare"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare      |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the link of folder "FolderToShare" from space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    Examples:
      | permissions-role |
      | View             |
      | Edit             |
      | Upload           |
      | File Drop        |
      | Secure View      |


  Scenario Outline: user removes access to a file in link share
    Given user "Alice" has uploaded file "filesForUpload/textfile.txt" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt       |
      | space           | Personal           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the link of file "textfile.txt" from space "Personal" using the Graph API
    Then the HTTP status code should be "204"
    Examples:
      | permissions-role |
      | View             |
      | Edit             |
      | Secure View      |


  Scenario Outline: user removes access to a folder in project space in link share
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has created the following resource link share:
      | resource        | FolderToShare      |
      | space           | NewSpace           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the link of folder "FolderToShare" from space "NewSpace" using the Graph API
    Then the HTTP status code should be "204"
    Examples:
      | permissions-role |
      | View             |
      | Edit             |
      | Upload           |
      | File Drop        |
      | Secure View      |


  Scenario Outline: user removes access to a file in project space in link share
    Given the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "some content" to "textfile.txt"
    And user "Alice" has created the following resource link share:
      | resource        | textfile.txt       |
      | space           | NewSpace           |
      | permissionsRole | <permissions-role> |
      | password        | %public%           |
    When user "Alice" removes the link of file "textfile.txt" from space "NewSpace" using the Graph API
    Then the HTTP status code should be "204"
    Examples:
      | permissions-role |
      | View             |
      | Edit             |
      | Secure View      |
