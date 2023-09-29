Feature: Tag
  As a user
  I want to tag resources
  So that I can sort and search them quickly

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "use-tag" with the default quota using the Graph API
    And user "Alice" has created a folder "folderMain" in space "use-tag"
    And user "Alice" has uploaded a file inside space "use-tag" with content "some content" to "folderMain/insideTheFolder.txt"


  Scenario: user creates tags for resources in the project space
    Given user "Alice" has shared a space "use-tag" with settings:
      | shareWith | Brian  |
      | role      | viewer |
    When user "Alice" creates the following tags for folder "folderMain" of space "use-tag":
      | tag level#1                    |
      | tag with symbols @^$#^%$@%!_+) |
    Then the HTTP status code should be "200"
    When user "Alice" sends PROPFIND request from the space "use-tag" to the resource "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value                                      |
      | oc:tags | tag level#1,tag with symbols @^$#^%$@%!_+) |
    When user "Alice" creates the following tags for file "folderMain/insideTheFolder.txt" of space "use-tag":
      | fileTag |
    Then the HTTP status code should be "200"
    When user "Brian" sends PROPFIND request from the space "use-tag" to the resource "folderMain/insideTheFolder.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value   |
      | oc:tags | fileTag |
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | tag level#1                    |
      | tag with symbols @^$#^%$@%!_+) |
      | fileTag                        |
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | tag level#1                    |
      | tag with symbols @^$#^%$@%!_+) |
      | fileTag                        |


  Scenario: user creates tags for resources in the personal space
    Given user "Alice" has created a folder "folderMain" in space "Alice Hansen"
    And user "Alice" has uploaded a file inside space "Alice Hansen" with content "some content" to "file.txt"
    When user "Alice" creates the following tags for folder "folderMain" of space "Alice Hansen":
      | my tag    |
      | important |
    Then the HTTP status code should be "200"
    When user "Alice" creates the following tags for file "file.txt" of space "Alice Hansen":
      | fileTag                       |
      | tag with symbol @^$#^%$@%!_+) |
    Then the HTTP status code should be "200"
    When user "Alice" sends PROPFIND request from the space "Alice Hansen" to the resource "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response to user "Alice" should contain a mountpoint "Alice Hansen" with these key and value pairs:
      | key     | value            |
      | oc:tags | my tag,important |
    When user "Alice" sends PROPFIND request from the space "Alice Hansen" to the resource "file.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response to user "Alice" should contain a mountpoint "Alice Hansen" with these key and value pairs:
      | key     | value                                 |
      | oc:tags | fileTag,tag with symbol @^$#^%$@%!_+) |
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | my tag                        |
      | important                     |
      | fileTag                       |
      | tag with symbol @^$#^%$@%!_+) |


  Scenario Outline: member of the space tries to create tag
    Given user "Alice" has shared a space "use-tag" with settings:
      | shareWith | Brian  |
      | role      | <role> |
    When user "Brian" creates the following tags for folder "folderMain/insideTheFolder.txt" of space "use-tag":
      | tag level#1                    |
      | tag with symbols @^$#^%$@%!_+) |
    Then the HTTP status code should be "<code>"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response <shouldOrNot> contain following tags:
      | tag level#1                    |
      | tag with symbols @^$#^%$@%!_+) |
    Examples:
      | role    | code | shouldOrNot |
      | viewer  | 403  | should not  |
      | editor  | 200  | should      |
      | manager | 200  | should      |


  Scenario: recipient has a created tags if share is accepted
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag |
      | marketing |
    And user "Alice" has created a share inside of space "use-tag" with settings:
      | path      | folderMain |
      | shareWith | Brian      |
      | role      | viewer     |
    When user "Brian" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain following tags:
      | folderTag |
      | marketing |
    When user "Brian" accepts share "/folderMain" offered by user "Alice" using the sharing API
    And user "Brian" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | folderTag |
      | marketing |


  Scenario Outline: recipient of the shared resource tries to create a tag
    Given user "Alice" has created a share inside of space "use-tag" with settings:
      | path      | folderMain |
      | shareWith | Brian      |
      | role      | <role>     |
    And user "Brian" has accepted share "/folderMain" offered by user "Alice"
    When user "Brian" creates the following tags for <resource> "<resourceName>" of space "Shares":
      | tag in a shared resource |
      | second tag               |
    Then the HTTP status code should be "<code>"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response <shouldOrNot> contain following tags:
      | tag in a shared resource |
      | second tag               |
    Examples:
      | role    | resource | resourceName                   | code | shouldOrNot |
      | viewer  | file     | folderMain/insideTheFolder.txt | 403  | should not  |
      | editor  | file     | folderMain/insideTheFolder.txt | 200  | should      |
      | manager | file     | folderMain/insideTheFolder.txt | 200  | should      |
      | viewer  | folder   | folderMain                     | 403  | should not  |
      | editor  | folder   | folderMain                     | 200  | should      |
      | manager | folder   | folderMain                     | 200  | should      |


  Scenario Outline: recipient of the shared resource tries to remove a tag
    Given user "Alice" has created a share inside of space "use-tag" with settings:
      | path      | folderMain |
      | shareWith | Brian      |
      | role      | <role>     |
    And user "Alice" has created the following tags for <resource> "<resourceName>" of the space "use-tag":
      | tag in a shared resource |
      | second tag               |
    And user "Brian" has accepted share "/folderMain" offered by user "Alice"
    When user "Brian" removes the following tags for <resource> "<resourceName>" of space "Shares":
      | tag in a shared resource |
      | second tag               |
    Then the HTTP status code should be "<code>"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response <shouldOrNot> contain following tags:
      | tag in a shared resource |
      | second tag               |
    Examples:
      | role    | resource | resourceName                   | code | shouldOrNot |
      | viewer  | file     | folderMain/insideTheFolder.txt | 403  | should      |
      | editor  | file     | folderMain/insideTheFolder.txt | 200  | should not  |
      | manager | file     | folderMain/insideTheFolder.txt | 200  | should not  |
      | viewer  | folder   | folderMain                     | 403  | should      |
      | editor  | folder   | folderMain                     | 200  | should not  |
      | manager | folder   | folderMain                     | 200  | should not  |


  Scenario: user removes folder tags
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag   |
      | marketing   |
      | development |
    When user "Alice" removes the following tags for folder "folderMain" of space "use-tag":
      | folderTag |
      | marketing |
    And user "Alice" sends PROPFIND request from the space "use-tag" to the resource "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value       |
      | oc:tags | development |


  Scenario: user lists tags after deleting some folder tags
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag   |
      | marketing   |
      | development |
    When user "Alice" removes the following tags for folder "folderMain" of space "use-tag":
      | folderTag |
      | marketing |
    Then the HTTP status code should be "200"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | development |
    And the response should not contain following tags:
      | folderTag |
      | marketing |


  Scenario: user lists the tags after deleting a folder
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag |
      | marketing |
    When user "Alice" removes the folder "folderMain" from space "use-tag"
    Then the HTTP status code should be "204"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain following tags:
      | folderTag |
      | marketing |


  Scenario: user lists the tags after deleting a space
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag |
      | marketing |
    And user "Alice" has disabled a space "use-tag"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain following tags:
      | folderTag |
      | marketing |
    When user "Alice" deletes a space "use-tag"
    Then the HTTP status code should be "204"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should not contain following tags:
      | folderTag |
      | marketing |


  Scenario: user lists the tags after restoring a deleted folder
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | folderTag |
      | marketing |
    And user "Alice" has removed the folder "folderMain" from space "use-tag"
    When user "Alice" restores the folder "folderMain" from the trash of the space "use-tag" to "/folderMain"
    Then the HTTP status code should be "201"
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | folderTag |
      | marketing |


  Scenario: user creates a comma-separated list of tags for resources in the project space
    Given user "Alice" has shared a space "use-tag" with settings:
      | shareWith | Brian  |
      | role      | viewer |
    When user "Alice" creates the following tags for folder "folderMain" of space "use-tag":
      | finance,नेपाल |
    Then the HTTP status code should be "200"
    When user "Alice" sends PROPFIND request from the space "use-tag" to the resource "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value         |
      | oc:tags | finance,नेपाल |
    When user "Alice" creates the following tags for file "folderMain/insideTheFolder.txt" of space "use-tag":
      | file,नेपाल,Tag |
    Then the HTTP status code should be "200"
    When user "Brian" sends PROPFIND request from the space "use-tag" to the resource "folderMain/insideTheFolder.txt" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value          |
      | oc:tags | file,नेपाल,Tag |
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | finance |
      | नेपाल   |
      | file    |
      | Tag     |


  Scenario: setting a comma-separated list of tags adds to any existing tags on the resource
    Given user "Alice" has created the following tags for folder "folderMain" of the space "use-tag":
      | finance,hr |
    When user "Alice" creates the following tags for folder "folderMain" of space "use-tag":
      | engineering,finance,qa |
    Then the HTTP status code should be "200"
    When user "Alice" sends PROPFIND request from the space "use-tag" to the resource "folderMain" using the WebDAV API
    Then the HTTP status code should be "207"
    And the "PROPFIND" response should contain a space "use-tag" with these key and value pairs:
      | key     | value                     |
      | oc:tags | engineering,finance,hr,qa |
    When user "Alice" lists all available tags via the Graph API
    Then the HTTP status code should be "200"
    And the response should contain following tags:
      | engineering |
      | finance     |
      | hr          |
      | qa          |
