Feature: Preview file in project space
  As a user
  I want to be able to download different files for the preview
  So that I can preview the thumbnail of the file

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "previews of the files" with the default quota using the Graph API
    And using spaces DAV path


  Scenario Outline: user can preview created txt files in the project space
    Given user "Alice" has uploaded a file inside space "previews of the files" with content "test" to "<entity>"
    When user "Alice" downloads the preview of "<entity>" of the space "previews of the files" with width "<width>" and height "<height>" using the WebDAV API
    Then the HTTP status code should be "200"
    Examples:
      | entity                | width | height |
      | /file.txt             | 36    | 36     |
      | /name with spaces.txt | 1200  | 1200   |


  Scenario Outline: user can preview image files in the project space
    Given using spaces DAV path
    And user "Alice" has uploaded a file from "<source>" to "<destination>" via TUS inside of the space "previews of the files" using the WebDAV API
    When user "Alice" downloads the preview of "<destination>" of the space "previews of the files" with width "<width>" and height "<height>" using the WebDAV API
    Then the HTTP status code should be "200"
    Examples:
      | source                        | destination    | width | height |
      | filesForUpload/testavatar.png | testavatar.png | 36    | 36     |
      | filesForUpload/testavatar.png | testavatar.png | 1200  | 1200   |
      | filesForUpload/testavatar.png | testavatar.png | 1920  | 1920   |
      | filesForUpload/testavatar.jpg | testavatar.jpg | 36    | 36     |
      | filesForUpload/testavatar.jpg | testavatar.jpg | 1200  | 1200   |
      | filesForUpload/testavatar.jpg | testavatar.jpg | 1920  | 1920   |
      | filesForUpload/example.gif    | example.gif    | 36    | 36     |
      | filesForUpload/example.gif    | example.gif    | 1200  | 1200   |
      | filesForUpload/example.gif    | example.gif    | 1280  | 1280   |
