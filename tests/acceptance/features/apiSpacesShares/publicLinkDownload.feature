Feature: Public can download folders from project space public link
  As a public
  I want to be able to download folder from public link
  So that I can gain access to it's contents

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "new-space" with the default quota using the Graph API


  Scenario: download a folder from public link of a space
    Given user "Alice" has created a folder "NewFolder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "NewFolder/test.txt"
    And user "Alice" has created a public link share of the space "new-space" with settings:
      | permissions | 1        |
      | name        | someName |
    When public downloads the folder "NewFolder" from the last created public link using the public files API
    Then the HTTP status code should be "200"
    And the downloaded tar archive should contain these files:
      | name               | content      |
      | NewFolder/test.txt | some content |

  @issue-5229
  Scenario: download a folder from public link of a folder inside a space
    Given user "Alice" has created a folder "NewFolder" in space "new-space"
    And user "Alice" has created a folder "NewFolder/folder" in space "new-space"
    And user "Alice" has uploaded a file inside space "new-space" with content "some content" to "NewFolder/folder/test.txt"
    And user "Alice" has created a public link share inside of space "new-space" with settings:
      | path        | NewFolder   |
      | shareType   | 3           |
      | permissions | 1           |
      | name        | public link |
    When public downloads the folder "folder" from the last created public link using the public files API
    Then the HTTP status code should be "200"
    And the downloaded tar archive should contain these files:
      | name            | content      |
      | folder/test.txt | some content |
