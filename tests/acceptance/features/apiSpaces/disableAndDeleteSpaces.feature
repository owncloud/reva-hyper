Feature: Disabling and deleting space
  As a manager of space
  I want to be able to disable the space first, then delete it.
  So that a disabled spaces isn't accessible by shared users.

  Note - this feature is run in CI with ACCOUNTS_HASH_DIFFICULTY set to the default for production
  See https://github.com/owncloud/ocis/issues/1542 and https://github.com/owncloud/ocis/pull/839

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
      | Bob      |
      | Carol    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "Project Moon" with the default quota using the Graph API
    And user "Alice" has shared a space "Project Moon" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And user "Alice" has shared a space "Project Moon" with settings:
      | shareWith | Bob    |
      | role      | viewer |


  Scenario Outline: user can disable their own space via the Graph API
    Given the administrator has assigned the role "<role>" to user "Alice" using the Graph API
    When user "Alice" disables a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Brian" should not have a space called "Project Moon"
    And the user "Bob" should not have a space called "Project Moon"
    Examples:
      | role        |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: user with role user and user light cannot disable other space via the Graph API
    Given the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    When user "Carol" tries to disable a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    And the user "Brian" should have a space called "Project Moon"
    And the user "Bob" should have a space called "Project Moon"
    Examples:
      | role       |
      | User       |
      | User Light |


  Scenario: a space manager can disable and delete space in which files and folders exist via the webDav API
    Given user "Alice" has uploaded a file inside space "Project Moon" with content "test" to "test.txt"
    And user "Alice" has created a folder "MainFolder" in space "Project Moon"
    When user "Alice" disables a space "Project Moon"
    Then the HTTP status code should be "204"
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"


  Scenario Outline: user cannot delete their own space without first disabling it
    Given the administrator has assigned the role "<role>" to user "Alice" using the Graph API
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "400"
    And the user "Alice" should have a space called "Project Moon"
    Examples:
      | role        |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: user can delete their own disabled space via the Graph API
    Given the administrator has assigned the role "<role>" to user "Alice" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Alice" deletes a space "Project Moon"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"
    Examples:
      | role        |
      | Admin       |
      | Space Admin |
      | User        |
      | User Light  |


  Scenario Outline: an admin and space manager can disable other space via the Graph API
    Given the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    When user "Carol" disables a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Carol" should not have a space called "Project Moon"
    Examples:
      | role        |
      | Admin       |
      | Space Admin |


  Scenario Outline: an admin and space manager can delete other disabled Space
    Given the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Carol" deletes a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "204"
    And the user "Alice" should not have a space called "Project Moon"
    And the user "Carol" should not have a space called "Project Moon"
    Examples:
      | role        |
      | Admin       |
      | Space Admin |


  Scenario Outline: user with role user and user light cannot delete others disabled space via the Graph API
    Given the administrator has assigned the role "<role>" to user "Carol" using the Graph API
    And user "Alice" has disabled a space "Project Moon"
    When user "Carol" tries to delete a space "Project Moon" owned by user "Alice"
    Then the HTTP status code should be "404"
    Examples:
      | role       |
      | User       |
      | User Light |
