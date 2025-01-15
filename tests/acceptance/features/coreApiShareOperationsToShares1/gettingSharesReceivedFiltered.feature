@skipOnReva
Feature: get the received shares filtered by type (user, group etc)
  As a user
  I want to filter the shares that I have received of a particular type (user, group etc)
  So that I can know about the status of the shares I've received

  Background:
    Given these users have been created with default attributes:
      | username |
      | Alice    |
      | Brian    |
    And group "grp1" has been created
    And user "Brian" has been added to group "grp1"
    And user "Alice" has created folder "/folderToShareWithUser"
    And user "Alice" has created folder "/folderToShareWithGroup"
    And user "Alice" has created folder "/folderToShareWithPublic"
    And user "Alice" has uploaded file with content "file to share with user" to "/fileToShareWithUser.txt"
    And user "Alice" has uploaded file with content "file to share with group" to "/fileToShareWithGroup.txt"
    And user "Alice" has uploaded file with content "file to share with public" to "/fileToShareWithPublic.txt"
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShareWithUser |
      | space           | Personal              |
      | sharee          | Brian                 |
      | shareType       | user                  |
      | permissionsRole | Viewer                |
    And user "Alice" has sent the following resource share invitation:
      | resource        | folderToShareWithGroup |
      | space           | Personal               |
      | sharee          | grp1                   |
      | shareType       | group                  |
      | permissionsRole | Viewer                 |
    And user "Alice" has created the following resource link share:
      | resource        | folderToShareWithPublic |
      | space           | Personal                |
      | permissionsRole | View                    |
      | password        | %public%                |
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShareWithUser.txt |
      | space           | Personal                |
      | sharee          | Brian                   |
      | shareType       | user                    |
      | permissionsRole | Viewer                  |
    And user "Alice" has sent the following resource share invitation:
      | resource        | fileToShareWithGroup.txt |
      | space           | Personal                 |
      | sharee          | grp1                     |
      | shareType       | group                    |
      | permissionsRole | Viewer                   |
    And user "Alice" has created the following resource link share:
      | resource        | fileToShareWithPublic.txt |
      | space           | Personal                  |
      | permissionsRole | View                      |
      | password        | %public%                  |


  Scenario Outline: getting shares received from users
    Given using OCS API version "<ocs-api-version>"
    When user "Brian" gets the user shares shared with him using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And exactly 2 files or folders should be included in the response
    And folder "/Shares/folderToShareWithUser" should be included in the response
    And file "/Shares/fileToShareWithUser.txt" should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |


  Scenario Outline: getting shares received from groups
    Given using OCS API version "<ocs-api-version>"
    When user "Brian" gets the group shares shared with him using the sharing API
    Then the OCS status code should be "<ocs-status-code>"
    And the HTTP status code should be "200"
    And exactly 2 files or folders should be included in the response
    And folder "/Shares/folderToShareWithGroup" should be included in the response
    And folder "/Shares/fileToShareWithGroup.txt" should be included in the response
    Examples:
      | ocs-api-version | ocs-status-code |
      | 1               | 100             |
      | 2               | 200             |
