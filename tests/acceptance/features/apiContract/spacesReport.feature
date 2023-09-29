Feature: REPORT request to project space
  As a user
  I want to check the REPORT response of project spaces
  So that I can make sure that the response contains all the relevant details

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "findData" with the default quota using the Graph API


  Scenario: check the response of the searched file
    Given user "Alice" has uploaded a file inside space "findData" with content "some content" to "testFile.txt"
    And using new DAV path
    When user "Alice" searches for "testFile.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | /testFile.txt |
    And the following headers should match these regular expressions
      | X-Request-Id | /^[a-zA-Z]+\/[a-zA-Z]+\.feature:\d+(-\d+)?$/ |
    And the "REPORT" response to user "Alice" should contain a mountpoint "findData" with these key and value pairs:
      | key                | value               |
      | oc:fileid          | UUIDof:testFile.txt |
      | oc:file-parent     | UUIDof:findData     |
      | oc:name            | testFile.txt        |
      | d:getcontenttype   | text/plain          |
      | oc:permissions     | RDNVW               |
      | d:getcontentlength | 12                  |


  Scenario: check the response of the searched sub-file
    Given user "Alice" has created a folder "folderMain/SubFolder1/subFOLDER2" in space "findData"
    And user "Alice" has uploaded a file inside space "findData" with content "some content" to "folderMain/SubFolder1/subFOLDER2/insideTheFolder.txt"
    And using new DAV path
    When user "Alice" searches for "insideTheFolder.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | /folderMain/SubFolder1/subFOLDER2/insideTheFolder.txt |
    And the following headers should match these regular expressions
      | X-Request-Id | /^[a-zA-Z]+\/[a-zA-Z]+\.feature:\d+(-\d+)?$/ |
    And the "REPORT" response to user "Alice" should contain a mountpoint "findData" with these key and value pairs:
      | key                | value                                                       |
      | oc:fileid          | UUIDof:folderMain/SubFolder1/subFOLDER2/insideTheFolder.txt |
      | oc:file-parent     | UUIDof:folderMain/SubFolder1/subFOLDER2                     |
      | oc:name            | insideTheFolder.txt                                         |
      | d:getcontenttype   | text/plain                                                  |
      | oc:permissions     | RDNVW                                                       |
      | d:getcontentlength | 12                                                          |


  Scenario: check the response of the searched folder
    Given user "Alice" has created a folder "folderMain" in space "findData"
    And using new DAV path
    When user "Alice" searches for "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | /folderMain |
    And the following headers should match these regular expressions
      | X-Request-Id | /^[a-zA-Z]+\/[a-zA-Z]+\.feature:\d+(-\d+)?$/ |
    And the "REPORT" response to user "Alice" should contain a mountpoint "findData" with these key and value pairs:
      | key              | value                |
      | oc:fileid        | UUIDof:folderMain    |
      | oc:file-parent   | UUIDof:findData      |
      | oc:name          | folderMain           |
      | d:getcontenttype | httpd/unix-directory |
      | oc:permissions   | RDNVCK               |
      | oc:size          | 0                    |


  Scenario: check the response of the searched sub-folder
    Given user "Alice" has created a folder "folderMain/sub-folder" in space "findData"
    And using new DAV path
    When user "Alice" searches for "*sub*" using the WebDAV API
    Then the HTTP status code should be "207"
    And the search result of user "Alice" should contain only these entries:
      | /folderMain/sub-folder |
    And the following headers should match these regular expressions
      | X-Request-Id | /^[a-zA-Z]+\/[a-zA-Z]+\.feature:\d+(-\d+)?$/ |
    Then the HTTP status code should be "207"
    And the "REPORT" response to user "Alice" should contain a mountpoint "findData" with these key and value pairs:
      | key              | value                        |
      | oc:fileid        | UUIDof:folderMain/sub-folder |
      | oc:file-parent   | UUIDof:folderMain            |
      | oc:name          | sub-folder                   |
      | d:getcontenttype | httpd/unix-directory         |
      | oc:permissions   | RDNVCK                       |
      | oc:size          | 0                            |
