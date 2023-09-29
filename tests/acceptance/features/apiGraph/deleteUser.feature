Feature: delete user
  As an admin
  I want to be able to delete users
  So that I can remove unnecessary users

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given user "Alice" has been created with default attributes and without skeleton files


  Scenario Outline: admin user deletes a user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the user "Alice" has created a new user with the following attributes using the Graph API:
      | userName    | <userName>    |
      | displayName | <displayName> |
      | email       | <email>       |
      | password    | <password>    |
    When the user "Alice" deletes a user "<userName>" using the Graph API
    Then the HTTP status code should be "204"
    And user "<userName>" should not exist
    Examples:
      | userName             | displayName     | email               | password                     |
      | SameDisplayName      | Alice Hansen    | new@example.org     | containsCharacters(*:!;_+-&) |
      | withoutPassSameEmail | without pass    | alice@example.org   |                              |
      | name                 | pass with space | example@example.org | my pass                      |


  Scenario: delete a user and specify the user name in different case
    Given user "brand-new-user" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" deletes a user "Brand-New-User" using the Graph API
    Then the HTTP status code should be "204"
    And user "brand-new-user" should not exist


  Scenario Outline: admin user deletes another user with different role
    Given user "Brian" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    When the user "Alice" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "204"
    And user "Brian" should not exist
    Examples:
      | role        |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin user tries to delete his/her own account
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" deletes a user "Alice" using the Graph API
    Then the HTTP status code should be "403"
    And user "Alice" should exist


  Scenario Outline: non-admin user tries to delete his/her own account
    Given the administrator has assigned the role "<role>" to user "Alice" using the Graph API
    When the user "Alice" deletes a user "Alice" using the Graph API
    Then the HTTP status code should be "401"
    And user "Alice" should exist
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario: admin user tries to delete a nonexistent user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    When the user "Alice" tries to delete a nonexistent user using the Graph API
    Then the HTTP status code should be "404"


  Scenario Outline: non-admin user tries to delete a nonexistent user
    Given the administrator has assigned the role "<role>" to user "Alice" using the Graph API
    When the user "Alice" tries to delete a nonexistent user using the Graph API
    Then the HTTP status code should be "401"
    Examples:
      | role        |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: non-admin user tries to delete another user with different role
    Given user "Brian" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    And the administrator has assigned the role "<userRole>" to user "Alice" using the Graph API
    When the user "Alice" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "401"
    And user "Brian" should exist
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


  Scenario: admin user deletes a disabled user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes and without skeleton files
    And the user "Alice" has disabled user "Brian" using the Graph API
    When the user "Alice" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "204"
    And user "Brian" should not exist


  Scenario Outline: normal user tries to delete a disabled user
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes and without skeleton files
    And user "Carol" has been created with default attributes and without skeleton files
    And the administrator has assigned the role "<role>" to user "Brian" using the Graph API
    And the administrator has assigned the role "<userRole>" to user "Carol" using the Graph API
    And the user "Alice" has disabled user "Brian" using the Graph API
    When the user "Carol" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "401"
    And user "Brian" should exist
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


  Scenario: personal space is deleted automatically when the user is deleted
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes and without skeleton files
    When the user "Alice" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "204"
    When user "Alice" lists all spaces via the Graph API with query "$filter=driveType eq 'personal'"
    Then the json responded should not contain a space with name "Brian Murphy"


  Scenario: accepted share is deleted automatically when the user is deleted
    Given the administrator has assigned the role "Admin" to user "Alice" using the Graph API
    And user "Brian" has been created with default attributes and without skeleton files
    And user "Brian" has created a folder "new" in space "Brian Murphy"
    And user "Brian" has created a share inside of space "Brian Murphy" with settings:
      | path      | new    |
      | shareWith | Alice  |
      | role      | viewer |
    And user "Alice" has accepted share "/new" offered by user "Brian"
    When the user "Alice" deletes a user "Brian" using the Graph API
    Then the HTTP status code should be "204"
    And as "Alice" folder "Shares/new" should not exist
