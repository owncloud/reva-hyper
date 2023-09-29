Feature: move (rename) file
  As a user
  I want to be able to move and rename files
  So that I can manage my file system

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path


  Scenario Outline: moving a file within same space project with role manager and editor
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "newfolder" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | <role> |
    When user "Alice" moves file "insideSpace.txt" to "newfolder/insideSpace.txt" in space "Project" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" the space "Project" should contain these entries:
      | newfolder/insideSpace.txt |
    And for user "Alice" the space "Project" should not contain these entries:
      | insideSpace.txt |
    Examples:
      | role    |
      | manager |
      | editor  |


  Scenario: moving a file within same space project with role viewer
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the Graph API
    And user "Brian" has created a folder "newfolder" in space "Project"
    And user "Brian" has uploaded a file inside space "Project" with content "some content" to "insideSpace.txt"
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | viewer |
    When user "Alice" moves file "insideSpace.txt" to "newfolder/insideSpace.txt" in space "Project" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Project" should not contain these entries:
      | newfolder/insideSpace.txt |
    And for user "Alice" the space "Project" should contain these entries:
      | insideSpace.txt |


  Scenario Outline: user moves a file from a space project with different a role to a space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project1" with the default quota using the Graph API
    And user "Brian" has created a space "Project2" with the default quota using the Graph API
    And user "Brian" has uploaded a file inside space "Project1" with content "Project1 content" to "project1.txt"
    And user "Brian" has shared a space "Project2" with settings:
      | shareWith | Alice     |
      | role      | <to_role> |
    And user "Brian" has shared a space "Project1" with settings:
      | shareWith | Alice       |
      | role      | <from_role> |
    When user "Alice" moves file "project1.txt" from space "Project1" to "project1.txt" inside space "Project2" using the WebDAV API
    Then the HTTP status code should be "<https_status_code>"
    And for user "Alice" the space "Project1" should contain these entries:
      | project1.txt |
    And for user "Alice" the space "Project2" should not contain these entries:
      | project1.txt |
    Examples:
      | from_role | to_role | https_status_code |
      | manager   | manager | 502               |
      | editor    | manager | 502               |
      | manager   | editor  | 502               |
      | editor    | editor  | 502               |
      | manager   | viewer  | 403               |
      | editor    | viewer  | 403               |
      | viewer    | manager | 403               |
      | viewer    | editor  | 403               |
      | viewer    | viewer  | 403               |


  Scenario Outline: user moves a file from a space project with different role to a space personal
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the GraphApi
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "project.txt"
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | <role> |
    When user "Alice" moves file "project.txt" from space "Project" to "project.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "<https_status_code>"
    And for user "Alice" the space "Project" should contain these entries:
      | project.txt |
    And for user "Alice" the space "Personal" should not contain these entries:
      | project.txt |
    Examples:
      | role    | https_status_code |
      | manager | 502               |
      | editor  | 502               |
      | viewer  | 403               |


  Scenario Outline: user moves a file from space project with different role to space Shares with different role (permission)
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the GraphApi
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded a file inside space "Project" with content "Project content" to "project.txt"
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | <role> |
    And user "Brian" has shared folder "/testshare" with user "Alice" with permissions "<permissions>"
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    When user "Alice" moves file "project.txt" from space "Project" to "/testshare/project.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Project" should contain these entries:
      | project.txt |
    And for user "Alice" the space "Shares" should not contain these entries:
      | /testshare/project.txt |
    Examples:
      | role    | permissions |
      | manager | 31          |
      | editor  | 31          |
      | manager | 17          |
      | editor  | 17          |
      | viewer  | 31          |
      | viewer  | 17          |


  Scenario Outline: user moves a file from space personal to space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the GraphApi
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | <role> |
    And user "Alice" has uploaded file with content "personal space content" to "/personal.txt"
    When user "Alice" moves file "personal.txt" from space "Personal" to "personal.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "<https_status_code>"
    And for user "Alice" the space "Personal" should contain these entries:
      | personal.txt |
    And for user "Alice" the space "Project" should not contain these entries:
      | personal.txt |
    Examples:
      | role    | https_status_code |
      | manager | 502               |
      | editor  | 502               |
      | viewer  | 403               |


  Scenario Outline: user moves a file from space personal to space Shares with different role (permission)
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has shared folder "/testshare" with user "Alice" with permissions "<permissions>"
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    And user "Alice" has uploaded file with content "personal content" to "personal.txt"
    When user "Alice" moves file "personal.txt" from space "Personal" to "/testshare/personal.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Personal" should contain these entries:
      | personal.txt |
    And for user "Alice" the space "Shares" should not contain these entries:
      | /testshare/personal.txt |
    Examples:
      | permissions |
      | 31          |
      | 17          |
      | 1           |


  Scenario Outline: user moves a file from space Shares with different role (permissions) to space personal
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has shared folder "/testshare" with user "Alice" with permissions "<permissions>"
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    When user "Alice" moves file "/testshare/testshare.txt" from space "Shares" to "testshare.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Personal" should not contain these entries:
      | testshare.txt |
    And for user "Alice" folder "testshare" of the space "Shares" should contain these entries:
      | testshare.txt |
    Examples:
      | permissions |
      | 31          |
      | 17          |
      | 1           |


  Scenario Outline: user moves a file from space Shares with different role (permissions) to space project with different role
    Given the administrator has assigned the role "Space Admin" to user "Brian" using the Graph API
    And user "Brian" has created a space "Project" with the default quota using the GraphApi
    And user "Brian" has shared a space "Project" with settings:
      | shareWith | Alice  |
      | role      | <role> |
    And user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "testshare content" to "/testshare/testshare.txt"
    And user "Brian" has shared folder "/testshare" with user "Alice" with permissions "<permissions>"
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    When user "Alice" moves file "/testshare/testshare.txt" from space "Shares" to "testshare.txt" inside space "Project" using the WebDAV API
    Then the HTTP status code should be "502"
    And for user "Alice" the space "Project" should not contain these entries:
      | /testshare.txt |
    And for user "Alice" folder "testshare" of the space "Shares" should contain these entries:
      | testshare.txt |
    Examples:
      | role    | permissions |
      | manager | 31          |
      | editor  | 31          |
      | viewer  | 31          |
      | manager | 17          |
      | editor  | 17          |
      | viewer  | 17          |


  Scenario: user moves a file from space Shares with role editor to space Shares with role editor
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has shared folder "/testshare1" with user "Alice" with permissions "31"
    And user "Brian" has shared folder "/testshare2" with user "Alice" with permissions "31"
    And user "Alice" has accepted share "/testshare1" offered by user "Brian"
    And user "Alice" has accepted share "/testshare2" offered by user "Brian"
    When user "Alice" moves file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Alice" folder "testshare2" of the space "Shares" should contain these entries:
      | testshare1.txt |
    And for user "Alice" folder "testshare1" of the space "Shares" should not contain these entries:
      | testshare1.txt |
    And for user "Brian" the space "Personal" should contain these entries:
      | /testshare2/testshare1.txt |


  Scenario: user moves a file from space Shares with role editor to space Shares with role viewer
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has shared folder "/testshare1" with user "Alice" with permissions "31"
    And user "Brian" has shared folder "/testshare2" with user "Alice" with permissions "17"
    And user "Alice" has accepted share "/testshare1" offered by user "Brian"
    And user "Alice" has accepted share "/testshare2" offered by user "Brian"
    When user "Alice" moves file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Shares" should not contain these entries:
      | /testshare2/testshare1.txt |
    And for user "Brian" the space "Personal" should not contain these entries:
      | /testshare2/testshare1.txt |


  Scenario: user moves a file from space Shares with role viewer to space Shares with role editor
    Given user "Brian" has created folder "/testshare1"
    And user "Brian" has created folder "/testshare2"
    And user "Brian" has uploaded file with content "testshare1 content" to "/testshare1/testshare1.txt"
    And user "Brian" has shared folder "/testshare1" with user "Alice" with permissions "17"
    And user "Brian" has shared folder "/testshare2" with user "Alice" with permissions "31"
    And user "Alice" has accepted share "/testshare1" offered by user "Brian"
    And user "Alice" has accepted share "/testshare2" offered by user "Brian"
    When user "Alice" moves file "/testshare1/testshare1.txt" from space "Shares" to "/testshare2/testshare1.txt" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "403"
    And for user "Alice" the space "Shares" should not contain these entries:
      | /testshare2/testshare1.txt |
    And for user "Brian" the space "Personal" should not contain these entries:
      | /testshare2/testshare1.txt |


  Scenario: checking file id after a move between received shares
    Given user "Alice" has created the following folders
      | path     |
      | /folderA |
      | /folderB |
    And user "Alice" has shared folder "/folderA" with user "Brian"
    And user "Alice" has shared folder "/folderB" with user "Brian"
    And user "Brian" has accepted share "/folderA" offered by user "Alice"
    And user "Brian" has accepted share "/folderB" offered by user "Alice"
    And user "Brian" has created a folder "/folderA/ONE" in space "Shares"
    And user "Brian" has created a folder "/folderA/ONE/TWO" in space "Shares"
    And user "Brian" has stored id of folder "/folderA/ONE" of the space "Shares"
    When user "Brian" moves folder "/folderA/ONE" from space "Shares" to "/folderB/ONE" inside space "Shares" using the WebDAV API
    Then the HTTP status code should be "201"
    And for user "Brian" the space "Shares" should contain these entries:
      | /folderA |
    And for user "Brian" folder "folderB" of the space "Shares" should contain these entries:
      | /ONE |
    And for user "Brian" folder "folderA" of the space "Shares" should not contain these entries:
      | /ONE     |
      | /ONE/TWO |
    And user "Brian" folder "/folderB/ONE" of the space "Shares" should have the previously stored id


  Scenario: moving a file out of a shared folder as a sharer
    Given user "Brian" has created folder "/testshare"
    And user "Brian" has uploaded file with content "test data" to "/testshare/testfile.txt"
    And user "Brian" has created a share with settings
      | path        | testshare |
      | shareType   | user      |
      | permissions | change    |
      | shareWith   | Alice     |
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    When user "Brian" moves file "/testshare/testfile.txt" from space "Personal" to "/testfile.txt" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/testfile.txt" for user "Brian" should be "test data"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | testfile.txt |
    And for user "Brian" the space "Personal" should not contain these entries:
      | /testshare/testfile.txt |


  Scenario: moving a folder out of a shared folder as a sharer
    Given user "Brian" has created the following folders
      | path                     |
      | /testshare               |
      | /testshare/testsubfolder |
    And user "Brian" has uploaded file with content "test data" to "/testshare/testsubfolder/testfile.txt"
    And user "Brian" has created a share with settings
      | path        | testshare |
      | shareType   | user      |
      | permissions | change    |
      | shareWith   | Alice     |
    And user "Alice" has accepted share "/testshare" offered by user "Brian"
    When user "Brian" moves folder "/testshare/testsubfolder" from space "Personal" to "/testsubfolder" inside space "Personal" using the WebDAV API
    Then the HTTP status code should be "201"
    And the content of file "/testsubfolder/testfile.txt" for user "Brian" should be "test data"
    And for user "Alice" folder "testshare" of the space "Shares" should not contain these entries:
      | testsubfolder |
    And for user "Brian" the space "Personal" should not contain these entries:
      | /testshare/testsubfolder |


  Scenario: overwriting a file while moving
    Given user "Brian" has created folder "/folder"
    And user "Brian" has uploaded file with content "some content" to "/folder/testfile.txt"
    And user "Brian" has uploaded file with content "old data version 1" to "/testfile.txt"
    And user "Brian" has uploaded file with content "new data version 2" to "/testfile.txt"
    When user "Brian" overwrites file "/testfile.txt" from space "Personal" to "folder/testfile.txt" inside space "Personal" while moving using the WebDAV API
    Then the HTTP status code should be "204"
    And the content of file "/folder/testfile.txt" for user "Brian" should be "new data version 2"
    And for user "Brian" the space "Personal" should not contain these entries:
      | /testfile.txt |
    When user "Brian" downloads version of the file "/folder/testfile.txt" with the index "1" of the space "Personal" using the WebDAV API
    Then the HTTP status code should be "200"
    And the downloaded content should be "old data version 1"
    And as "Brian" file "testfile.txt" should exist in the trashbin of the space "Personal"
