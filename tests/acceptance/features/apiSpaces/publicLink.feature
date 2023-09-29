Feature: public link for a space

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "public space" with the default quota using the Graph API
    And user "Alice" has created a public link share of the space "public space" with settings:
      | permissions | 1 |


  Scenario: public tries to upload a file in the public space
    When the public uploads file "test.txt" with content "test" using the new public WebDAV API
    And the HTTP status code should be "403"


  Scenario: public tries to create a folder in the public space
    When the public creates folder "created-by-public" using the new public WebDAV API
    And the HTTP status code should be "403"


  Scenario: public tries to delete a file in the public space
    Given user "Alice" has uploaded a file inside space "public space" with content "some content" to "test.txt"
    When the public deletes file "test.txt" from the last public link share using the new public WebDAV API
    And the HTTP status code should be "403"


  Scenario: public tries to delete a folder in the public space
    And user "Alice" has created a folder "/public-folder" in space "public space"
    When the public deletes folder "public-folder" from the last public link share using the new public WebDAV API
    And the HTTP status code should be "403"


  Scenario: public tries to change content of a resources in the public space
    Given user "Alice" has uploaded a file inside space "public space" with content "some content" to "test.txt"
    When the public overwrites file "test.txt" with content "public content" using the new WebDAV API
    And the HTTP status code should be "403"
