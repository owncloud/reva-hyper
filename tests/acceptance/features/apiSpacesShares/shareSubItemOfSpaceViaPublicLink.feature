Feature: Share a file or folder that is inside a space via public link
      As a user with manager space role
      I want to be able to share the data inside the space via public link
      So that an anonymous user can have access to certain resources

      folder permissions:
      | role        | permissions |
      | internal    | 0           |
      | viewer      | 1           |
      | uploader    | 4           |
      | contributor | 5           |
      | editor      | 15          |

      file permissions:
      | role     | permissions |
      | internal | 0           |
      | viewer   | 1           |
      | editor   | 3           |

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
    And using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "share sub-item" with the default quota using the Graph API
    And user "Alice" has created a folder "folder" in space "share sub-item"
    And user "Alice" has uploaded a file inside space "share sub-item" with content "some content" to "folder/file.txt"

  @issue-5139
  Scenario Outline: manager of the space can share an entity inside project space via public link
    When user "Alice" creates a public link share inside of space "share sub-item" with settings:
      | path        | <entity>      |
      | shareType   | 3             |
      | permissions | <permissions> |
      | password    | <password>    |
      | name        | <name>        |
      | expireDate  | <expireDate>  |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Alice" and space "share sub-item" should include
      | item_type         | <item_type>   |
      | mimetype          | <mimetype>    |
      | file_target       | <file_target> |
      | path              | /<entity>     |
      | permissions       | <permissions> |
      | share_type        | public_link   |
      | displayname_owner | %displayname% |
      | name              | <name>        |
      | uid_file_owner    | %space_id%    |
      | space_id          | %space_id%    |
    Examples:
      | entity          | file_target | permissions | password | name | expireDate               | item_type | mimetype             |
      | folder          | /folder     | 0           |          | link |                          | folder    | httpd/unix-directory |
      | folder          | /folder     | 1           | 123      | link | 2042-03-25T23:59:59+0100 | folder    | httpd/unix-directory |
      | folder          | /folder     | 4           |          |      |                          | folder    | httpd/unix-directory |
      | folder          | /folder     | 5           | 200      |      | 2042-03-25T23:59:59+0100 | folder    | httpd/unix-directory |
      | folder          | /folder     | 15          |          | link |                          | folder    | httpd/unix-directory |
      | folder/file.txt | /file.txt   | 0           | 123      | link | 2042-03-25T23:59:59+0100 | file      | text/plain           |
      | folder/file.txt | /file.txt   | 1           | 123      | link | 2042-03-25T23:59:59+0100 | file      | text/plain           |
      | folder/file.txt | /file.txt   | 3           | 123      | link | 2042-03-25T23:59:59+0100 | file      | text/plain           |

  @issue-5139
  Scenario Outline: user participant of the project space with space manager role can share an entity inside project space via public link
    Given user "Alice" has shared a space "share sub-item" with settings:
      | shareWith | Brian   |
      | role      | manager |
    When user "Brian" creates a public link share inside of space "share sub-item" with settings:
      | path        | <entity>                 |
      | shareType   | 3                        |
      | permissions | 1                        |
      | password    | 123                      |
      | name        | public link              |
      | expireDate  | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "200"
    And the OCS status code should be "200"
    And the OCS status message should be "OK"
    And the fields of the last response to user "Brian" and space "share sub-item" should include
      | item_type         | <item_type>   |
      | mimetype          | <mimetype>    |
      | file_target       | <file_target> |
      | path              | /<entity>     |
      | share_type        | public_link   |
      | displayname_owner | %displayname% |
      | name              | public link   |
      | uid_file_owner    | %space_id%    |
      | space_id          | %space_id%    |
    Examples:
      | entity          | file_target | item_type | mimetype             |
      | folder          | /folder     | folder    | httpd/unix-directory |
      | folder/file.txt | /file.txt   | file      | text/plain           |

  @skipOnRevaMaster
  Scenario Outline: user participant of the project space without space manager role cannot share an entity inside project space via public link
    Given user "Alice" has shared a space "share sub-item" with settings:
      | shareWith | Brian       |
      | role      | <spaceRole> |
    When user "Brian" creates a public link share inside of space "share sub-item" with settings:
      | path        | <entity>                 |
      | shareType   | 3                        |
      | permissions | 1                        |
      | password    | 123                      |
      | name        | public link              |
      | expireDate  | 2042-03-25T23:59:59+0100 |
    Then the HTTP status code should be "403"
    And the OCS status code should be "403"
    And the OCS status message should be "No share permission"
    Examples:
      | entity          | spaceRole |
      | folder          | editor    |
      | folder          | viewer    |
      | folder/file.txt | editor    |
      | folder/file.txt | viewer    |


  Scenario Outline: user creates a new public link share of a file inside the personal space with edit permissions
    Given using OCS API version "<ocs_api_version>"
    And user "Alice" has uploaded file with content "Random data" to "/file.txt"
    When user "Alice" creates a public link share using the sharing API with settings
      | path        | file.txt    |
      | permissions | read,update |
    Then the OCS status code should be "<ocs_status_code>"
    And the HTTP status code should be "200"
    And the fields of the last response to user "Alice" should include
      | item_type              | file          |
      | mimetype               | text/plain    |
      | file_target            | /file.txt     |
      | path                   | /file.txt     |
      | permissions            | read,update   |
      | share_type             | public_link   |
      | displayname_file_owner | %displayname% |
      | displayname_owner      | %displayname% |
      | uid_file_owner         | %username%    |
      | uid_owner              | %username%    |
    And the public should be able to download the last publicly shared file using the new public WebDAV API without a password and the content should be "Random data"
    And the public upload to the last publicly shared file using the new public WebDAV API should pass with HTTP status code "204"
    Examples:
      | ocs_api_version | ocs_status_code |
      | 1               | 100             |
      | 2               | 200             |

  @issue-5139
  Scenario Outline: user participant of the project space can see the created public resources link
    Given user "Alice" has shared a space "share sub-item" with settings:
      | shareWith | Brian       |
      | role      | <spaceRole> |
    When user "Alice" creates a public link share inside of space "share sub-item" with settings:
      | path        | folder/file.txt |
      | shareType   | 3               |
      | permissions | 1               |
    Then the fields of the last response to user "Alice" and space "share sub-item" should include
      | item_type              | file             |
      | mimetype               | text/plain       |
      | file_target            | /file.txt        |
      | path                   | /folder/file.txt |
      | permissions            | 1                |
      | share_type             | public_link      |
      | displayname_file_owner |                  |
      | displayname_owner      | %displayname%    |
      | uid_owner              | %username%       |
      | uid_file_owner         | %space_id%       |
      | space_id               | %space_id%       |
    And for user "Brian" the space "share sub-item" should contain the last created public link of the file "folder/file.txt"
    Examples:
      | spaceRole |
      | editor    |
      | viewer    |
      | manager   |
