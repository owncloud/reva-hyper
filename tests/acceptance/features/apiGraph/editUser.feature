Feature: edit user
  As an admin
  I want to be able to edit user information
  So that I can manage users

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given user "Alice" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | Brian             |
      | displayName | Brian Murphy      |
      | email       | brian@example.com |
      | password    | 1234              |


  Scenario Outline: admin user can edit another user's email
    When the user "Alice" changes the email of user "Brian" to "<newEmail>" using the Graph API
    Then the HTTP status code should be "<code>"
    And the user information of "Brian" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "mail"
      ],
      "properties": {
        "mail": {
          "type": "string",
          "enum": ["<emailAsResult>"]
        }
      }
    }
    """
    Examples:
      | action description        | newEmail             | code | emailAsResult        |
      | change to a valid email   | newemail@example.com | 200  | newemail@example.com |
      | override existing mail    | brian@example.com    | 200  | brian@example.com    |
      | two users with same mail  | alice@example.org    | 200  | alice@example.org    |
      | empty mail                |                      | 400  | brian@example.com    |
      | change to a invalid email | invalidEmail         | 400  | brian@example.com    |

  @issue-7044
  Scenario Outline: admin user can edit another user's name
    Given user "Carol" has been created with default attributes and without skeleton files
    When the user "Alice" changes the user name of user "Carol" to "<userName>" using the Graph API
    Then the HTTP status code should be "<code>"
    And the user information of "<newUserName>" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "onPremisesSamAccountName"
      ],
      "properties": {
          "type": "string",
        "onPremisesSamAccountName": {
          "enum": ["<newUserName>"]
        }
      }
    }
    """
    Examples:
      | action description           | userName | code | newUserName |
      | change to a valid user name  | Lionel   | 200  | Lionel      |
      | user name characters         | a*!_+-&  | 200  | a*!_+-&     |
      | change to existing user name | Brian    | 409  | Brian       |
      | empty user name              |          | 400  | Brian       |


  Scenario: admin user changes the name of a user to the name of an existing disabled user
    Given the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | sam             |
      | displayName | sam             |
      | email       | sam@example.com |
      | password    | 1234            |
    And the user "Alice" has disabled user "Brian" using the Graph API
    When the user "Alice" changes the user name of user "sam" to "Brian" using the Graph API
    Then the HTTP status code should be "409"
    And the user information of "sam" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "onPremisesSamAccountName"
      ],
      "properties": {
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["sam"]
        }
      }
    }
    """


  Scenario: admin user changes the name of a user to the name of a previously deleted user
    Given the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | sam             |
      | displayName | sam             |
      | email       | sam@example.com |
      | password    | 1234            |
    And the user "Alice" has deleted a user "sam" using the Graph API
    When the user "Alice" changes the user name of user "Brian" to "sam" using the Graph API
    Then the HTTP status code should be "200"
    And the user information of "sam" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "onPremisesSamAccountName"
      ],
      "properties": {
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["sam"]
        }
      }
    }
    """


  Scenario Outline: normal user should not be able to change their email address
    Given the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When the user "Brian" tries to change the email of user "Brian" to "newemail@example.com" using the Graph API
    Then the HTTP status code should be "401"
    And the user information of "Brian" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "mail"
      ],
      "properties": {
        "mail": {
          "type": "string",
          "enum": ["brian@example.com"]
        }
      }
    }
    """
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: normal user should not be able to edit another user's email
    Given the administrator has assigned the role "<userRole>" to user "Brian" using the Graph API
    And the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | Carol             |
      | displayName | Carol King        |
      | email       | carol@example.com |
      | password    | 1234              |
    And the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    When the user "Brian" tries to change the email of user "Carol" to "newemail@example.com" using the Graph API
    Then the HTTP status code should be "401"
    And the user information of "Carol" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "mail"
      ],
      "properties": {
        "mail": {
          "type": "string",
          "enum": ["carol@example.com"]
        }
      }
    }
    """
    Examples:
      | userRole    | role        |
      | Space Admin | Space Admin |
      | Space Admin | User        |
      | Space Admin | User Light  |
      | Space Admin | Admin       |
      | User        | Space Admin |
      | User        | User        |
      | User        | User Light  |
      | User        | Admin       |
      | User Light  | Space Admin |
      | User Light  | User        |
      | User Light  | User Light  |
      | User Light  | Admin       |


  Scenario Outline: admin user can edit another user display name
    When the user "Alice" changes the display name of user "Brian" to "<newDisplayName>" using the Graph API
    Then the HTTP status code should be "200"
    And the user information of "Brian" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "displayName"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["<displayNameAsResult>"]
        }
      }
    }
    """
    Examples:
      | action description                | newDisplayName | displayNameAsResult |
      | change to a display name          | Olaf Scholz    | Olaf Scholz         |
      | override to existing display name | Carol King     | Carol King          |
      | change to an empty display name   |                | Brian Murphy        |
      | displayName with characters       | *:!;_+-&#(?)   | *:!;_+-&#(?)        |


  Scenario Outline: normal user should not be able to change his/her own display name
    Given the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When the user "Brian" tries to change the display name of user "Brian" to "Brian Murphy" using the Graph API
    Then the HTTP status code should be "401"
    And the user information of "Alice" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "displayName"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Alice Hansen"]
        }
      }
    }
    """
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: normal user should not be able to edit another user's display name
    Given the administrator has assigned the role "<userRole>" to user "Brian" using the Graph API
    And the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | Carol             |
      | displayName | Carol King        |
      | email       | carol@example.com |
      | password    | 1234              |
    And the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    When the user "Brian" tries to change the display name of user "Carol" to "Alice Hansen" using the Graph API
    Then the HTTP status code should be "401"
    And the user information of "Carol" should match this JSON schema
    """
    {
      "type": "object",
      "required": [
        "displayName"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Carol King"]
        }
      }
    }
    """
    Examples:
      | userRole    | role        |
      | Space Admin | Space Admin |
      | Space Admin | User        |
      | Space Admin | User Light  |
      | Space Admin | Admin       |
      | User        | Space Admin |
      | User        | User        |
      | User        | User Light  |
      | User        | Admin       |
      | User Light  | Space Admin |
      | User Light  | User        |
      | User Light  | User Light  |
      | User Light  | Admin       |


  Scenario: admin user resets password of another user
    Given user "Brian" has uploaded file with content "test file for reset password" to "/resetpassword.txt"
    When the user "Alice" resets the password of user "Brian" to "newpassword" using the Graph API
    Then the HTTP status code should be "200"
    And the content of file "resetpassword.txt" for user "Brian" using password "newpassword" should be "test file for reset password"


  Scenario Outline: normal user should not be able to reset the password of another user
    Given the administrator has assigned the role "<userRole>" to user "Brian" using the Graph API
    And the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | Carol             |
      | displayName | Carol King        |
      | email       | carol@example.com |
      | password    | 1234              |
    And the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    And user "Carol" has uploaded file with content "test file for reset password" to "/resetpassword.txt"
    When the user "Brian" resets the password of user "Carol" to "newpassword" using the Graph API
    Then the HTTP status code should be "401"
    And the content of file "resetpassword.txt" for user "Carol" using password "1234" should be "test file for reset password"
    But user "Carol" using password "newpassword" should not be able to download file "resetpassword.txt"
    Examples:
      | userRole    | role        |
      | Space Admin | Space Admin |
      | Space Admin | User        |
      | Space Admin | User Light  |
      | Space Admin | Admin       |
      | User        | Space Admin |
      | User        | User        |
      | User        | User Light  |
      | User        | Admin       |
      | User Light  | Space Admin |
      | User Light  | User        |
      | User Light  | User Light  |
      | User Light  | Admin       |


  Scenario: admin user disables another user
    When the user "Alice" disables user "Brian" using the Graph API
    Then the HTTP status code should be "200"
    When user "Alice" gets information of user "Brian" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "displayName",
        "id",
        "mail",
        "onPremisesSamAccountName",
        "accountEnabled"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Brian Murphy"]
        },
        "id" : {
          "type": "string",
          "pattern": "^%user_id_pattern%$"
        },
        "mail": {
          "type": "string",
          "enum": ["brian@example.com"]
        },
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["Brian"]
        },
        "accountEnabled": {
          "type": "boolean",
          "enum": [false]
        }
      }
    }
    """


  Scenario Outline: normal user should not be able to disable another user
    Given user "Carol" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When the user "Brian" tries to disable user "Carol" using the Graph API
    Then the HTTP status code should be "401"
    When user "Alice" gets information of user "Carol" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "displayName",
        "id",
        "mail",
        "onPremisesSamAccountName",
        "accountEnabled"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Carol King"]
        },
        "id" : {
          "type": "string",
          "pattern": "^%user_id_pattern%$"
        },
        "mail": {
          "type": "string",
          "enum": ["carol@example.org"]
        },
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["Carol"]
        },
        "accountEnabled": {
          "type": "boolean",
          "enum": [true]
        }
      }
    }
    """
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin user enables disabled user
    Given the user "Alice" has disabled user "Brian" using the Graph API
    When the user "Alice" enables user "Brian" using the Graph API
    Then the HTTP status code should be "200"
    When user "Alice" gets information of user "Brian" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "displayName",
        "id",
        "mail",
        "onPremisesSamAccountName",
        "accountEnabled"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Brian Murphy"]
        },
        "id" : {
          "type": "string",
          "pattern": "^%user_id_pattern%$"
        },
        "mail": {
          "type": "string",
          "enum": ["brian@example.com"]
        },
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["Brian"]
        },
        "accountEnabled": {
          "type": "boolean",
          "enum": [true]
        }
      }
    }
    """


  Scenario Outline: normal user should not be able to enable another user
    Given user "Carol" has been created with default attributes and without skeleton files
    And the user "Alice" has disabled user "Carol" using the Graph API
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When the user "Brian" tries to enable user "Carol" using the Graph API
    Then the HTTP status code should be "401"
    When user "Alice" gets information of user "Carol" using Graph API
    Then the HTTP status code should be "200"
    And the JSON data of the response should match
    """
    {
      "type": "object",
      "required": [
        "displayName",
        "id",
        "mail",
        "onPremisesSamAccountName",
        "accountEnabled"
      ],
      "properties": {
        "displayName": {
          "type": "string",
          "enum": ["Carol King"]
        },
        "id" : {
          "type": "string",
          "pattern": "^%user_id_pattern%$"
        },
        "mail": {
          "type": "string",
          "enum": ["carol@example.org"]
        },
        "onPremisesSamAccountName": {
          "type": "string",
          "enum": ["Carol"]
        },
        "accountEnabled": {
          "type": "boolean",
          "enum": [false]
        }
      }
    }
    """
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |
