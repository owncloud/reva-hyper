Feature: Update permission of a share
  As a user
  I want to update resources shared with me
  So that I can have more control over my shares and manage it
  https://owncloud.dev/libre-graph-api/#/drives.permissions/UpdatePermission

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |


  Scenario: user updates expiration date of a share
    Given user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has sent the following share invitation:
      | resource           | testfile.txt         |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
      | expirationDateTime | 2025-07-15T14:00:00Z |
    When user "Alice" updates the last share with the following using the Graph API:
      | space              | Personal             |
      | resource           | testfile.txt         |
      | expirationDateTime | 2200-07-15T14:00:00Z |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "expirationDateTime",
        "grantedToV2",
        "id",
        "roles"
      ],
      "properties": {
        "expirationDateTime": {
          "type": "string",
          "enum": ["2200-07-15T14:00:00Z"]
        },
        "grantedToV2": {
          "type": "object",
          "required": [
            "user"
          ],
          "properties":{
            "user": {
              "type": "object",
              "required": [
                "displayName",
                "id"
              ],
              "properties": {
                "displayName": {
                  "type": "string",
                  "enum": ["Brian Murphy"]
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                }
              }
            }
          }
        },
        "id": {
          "type": "string",
          "pattern": "^%permissions_id_pattern%$"
        },
        "roles": {
          "type": "array",
          "minItems": 1,
          "maxItems": 1,
          "items": {
            "type": "string",
            "pattern": "^%role_id_pattern%$"
          }
        }
      }
    }
    """


  Scenario Outline: user removes expiration date of a share
    Given user "Alice" has uploaded file with content "hello world" to "testfile.txt"
    And user "Alice" has created folder "folder"
    And user "Alice" has sent the following share invitation:
      | resource           | <resource>           |
      | space              | Personal             |
      | sharee             | Brian                |
      | shareType          | user                 |
      | permissionsRole    | Viewer               |
      | expirationDateTime | 2025-07-15T14:00:00Z |
    When user "Alice" updates the last share with the following using the Graph API:
      | space              | Personal   |
      | resource           | <resource> |
      | expirationDateTime |            |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "grantedToV2",
        "id",
        "roles"
      ],
      "properties": {
        "grantedToV2": {
          "type": "object",
          "required": [
            "user"
          ],
          "properties":{
            "user": {
              "type": "object",
              "required": [
                "displayName",
                "id"
              ],
              "properties": {
                "displayName": {
                  "type": "string",
                  "enum": ["Brian Murphy"]
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                }
              }
            }
          }
        },
        "id": {
          "type": "string",
          "pattern": "^%permissions_id_pattern%$"
        },
        "roles": {
          "type": "array",
          "minItems": 1,
          "maxItems": 1,
          "items": {
            "type": "string",
            "pattern": "^%role_id_pattern%$"
          }
        }
      }
    }
    """
    Examples:
      | resource     |
      | testfile.txt |
      | folder       |


  Scenario Outline: user updates role of a share
    Given user "Alice" has uploaded file with content "to share" to "/textfile1.txt"
    And user "Alice" has created folder "FolderToShare"
    And user "Alice" has sent the following share invitation:
      | resource        | <resource>         |
      | space           | Personal           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last share with the following using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | Personal               |
      | resource        | <resource>             |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "grantedToV2",
        "id",
        "roles"
      ],
      "properties": {
        "grantedToV2": {
          "type": "object",
          "required": [
            "user"
          ],
          "properties":{
            "user": {
              "type": "object",
              "required": [
                "displayName",
                "id"
              ],
              "properties": {
                "displayName": {
                  "type": "string",
                  "enum": ["Brian Murphy"]
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                }
              }
            }
          }
        },
        "id": {
          "type": "string",
          "pattern": "^%permissions_id_pattern%$"
        },
        "roles": {
          "type": "array",
          "minItems": 1,
          "maxItems": 1,
          "items": {
            "type": "string",
            "pattern": "^%role_id_pattern%$"
          }
        }
      }
    }
    """
    Examples:
      | permissions-role | resource      | new-permissions-role |
      | Viewer           | textfile1.txt | File Editor          |
      | File Editor      | textfile1.txt | Viewer               |
      | Viewer           | FolderToShare | Uploader             |
      | Viewer           | FolderToShare | Editor               |
      | Editor           | FolderToShare | Viewer               |
      | Editor           | FolderToShare | Uploader             |
      | Uploader         | FolderToShare | Editor               |
      | Uploader         | FolderToShare | Viewer               |


  Scenario Outline: space admin updates role of a member in project space (permissions endpoint)
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has sent the following share invitation:
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last share with the following using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "grantedToV2",
        "id",
        "roles"
      ],
      "properties": {
        "grantedToV2": {
          "type": "object",
          "required": [
            "user"
          ],
          "properties":{
            "user": {
              "type": "object",
              "required": [
                "displayName",
                "id"
              ],
              "properties": {
                "displayName": {
                  "const": "Brian Murphy"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                }
              }
            }
          }
        },
        "id": {
          "type": "string",
          "pattern": "^u:%user_id_pattern%$"
        },
        "roles": {
          "type": "array",
          "minItems": 1,
          "maxItems": 1,
          "items": {
            "type": "string",
            "pattern": "^%role_id_pattern%$"
          }
        }
      }
    }
    """
    Examples:
      | permissions-role | new-permissions-role |
      | Space Viewer     | Space Editor         |
      | Space Viewer     | Manager              |
      | Space Editor     | Space Viewer         |
      | Space Editor     | Manager              |
      | Manager          | Space Editor         |
      | Manager          | Space Viewer         |


  Scenario Outline: user updates role of a shared resource of project space
    Given using spaces DAV path
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "NewSpace" with the default quota using the Graph API
    And user "Alice" has uploaded a file inside space "NewSpace" with content "share space items" to "textfile1.txt"
    And user "Alice" has created a folder "FolderToShare" in space "NewSpace"
    And user "Alice" has sent the following share invitation:
      | resource        | <resource>         |
      | space           | NewSpace           |
      | sharee          | Brian              |
      | shareType       | user               |
      | permissionsRole | <permissions-role> |
    When user "Alice" updates the last share with the following using the Graph API:
      | permissionsRole | <new-permissions-role> |
      | space           | NewSpace               |
      | resource        | <resource>             |
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "grantedToV2",
        "id",
        "roles"
      ],
      "properties": {
        "grantedToV2": {
          "type": "object",
          "required": [
            "user"
          ],
          "properties":{
            "user": {
              "type": "object",
              "required": [
                "displayName",
                "id"
              ],
              "properties": {
                "displayName": {
                  "const": "Brian Murphy"
                },
                "id": {
                  "type": "string",
                  "pattern": "^%user_id_pattern%$"
                }
              }
            }
          }
        },
        "id": {
          "type": "string",
          "pattern": "^%permissions_id_pattern%$"
        },
        "roles": {
          "type": "array",
          "minItems": 1,
          "maxItems": 1,
          "items": {
            "type": "string",
            "pattern": "^%role_id_pattern%$"
          }
        }
      }
    }
    """
    Examples:
      | permissions-role | new-permissions-role | resource      |
      | Viewer           | File Editor          | textfile1.txt |
      | File Editor      | Viewer               | textfile1.txt |
      | Viewer           | Editor               | FolderToShare |
      | Viewer           | Uploader             | FolderToShare |
      | Editor           | Viewer               | FolderToShare |
      | Editor           | Uploader             | FolderToShare |
      | Uploader         | Viewer               | FolderToShare |
      | Uploader         | Editor               | FolderToShare |