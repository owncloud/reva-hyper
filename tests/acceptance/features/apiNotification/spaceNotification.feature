Feature: Notification
  As a user
  I want to be notified of actions related to space
  So that I can stay updated about the spaces

  Background:
    Given these users have been created with default attributes and without skeleton files:
      | username |
      | Alice    |
      | Brian    |
      | Carol    |
    And the administrator has assigned the role "Space Admin" to user "Alice" using the Graph API
    And user "Alice" has created a space "notification checking" with the default quota using the Graph API


  Scenario: get a notification of space shared
    Given user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the JSON response should contain a notification message with the subject "Space shared" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "app",
          "datetime",
          "message",
          "messageRich",
          "messageRichParameters",
          "notification_id",
          "object_id",
          "object_type",
          "subject",
          "subjectRich",
          "user"
        ],
        "properties": {
          "app": {
            "type": "string",
            "enum": [
              "userlog"
            ]
          },
          "message": {
            "type": "string",
            "enum": [
              "Alice Hansen added you to Space notification checking"
            ]
          },
          "messageRich": {
            "type": "string",
            "enum": [
              "{user} added you to Space {space}"
            ]
          },
          "messageRichParameters": {
            "type": "object",
            "required": [
              "space",
              "user"
            ],
            "properties": {
              "space": {
                "type": "object",
                "required": [
                  "id",
                  "name"
                ],
                "properties": {
                  "id": {
                    "type": "string",
                    "pattern": "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}\\$[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}![a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "notification checking"
                    ]
                  }
                }
              },
              "user": {
                "type": "object",
                "required": [
                  "displayname",
                  "id",
                  "name"
                ],
                "properties": {
                  "displayname": {
                    "type": "string",
                    "enum": [
                      "Alice Hansen"
                    ]
                  },
                  "id": {
                    "type": "string",
                    "enim": [
                      "%user_id%"
                    ]
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "Alice"
                    ]
                  }
                }
              }
            }
          },
          "notification_id": {
            "type": "string"
          },
          "object_id": {
            "type": "string",
            "pattern": "^%space_id_pattern%$"
          },
          "object_type": {
            "type": "string",
            "enum": [
              "storagespace"
            ]
          },
          "subject": {
            "type": "string",
            "enum": [
              "Space shared"
            ]
          },
          "subjectRich": {
            "type": "string",
            "enum": [
              "Space shared"
            ]
          },
          "user": {
            "type": "string",
            "enum": [
              "Alice"
            ]
          }
        }
      }
      """


  Scenario: get a notification of space unshared
    Given user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And user "Alice" has unshared a space "notification checking" shared with "Brian"
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the JSON response should contain a notification message with the subject "Removed from Space" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "app",
          "datetime",
          "message",
          "messageRich",
          "messageRichParameters",
          "notification_id",
          "object_id",
          "object_type",
          "subject",
          "subjectRich",
          "user"
        ],
        "properties": {
          "app": {
            "type": "string",
            "enum": [
              "userlog"
            ]
          },
          "message": {
            "type": "string",
            "enum": [
              "Alice Hansen removed you from Space notification checking"
            ]
          },
          "messageRich": {
            "type": "string",
            "enum": [
              "{user} removed you from Space {space}"
            ]
          },
          "messageRichParameters": {
            "type": "object",
            "required": [
              "space",
              "user"
            ],
            "properties": {
              "space": {
                "type": "object",
                "required": [
                  "id",
                  "name"
                ],
                "properties": {
                  "id": {
                    "type": "string",
                    "pattern": "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}\\$[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}![a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "notification checking"
                    ]
                  }
                }
              },
              "user": {
                "type": "object",
                "required": [
                  "displayname",
                  "id",
                  "name"
                ],
                "properties": {
                  "displayname": {
                    "type": "string",
                    "enum": [
                      "Alice Hansen"
                    ]
                  },
                  "id": {
                    "type": "string",
                    "enim": [
                      "%user_id%"
                    ]
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "Alice"
                    ]
                  }
                }
              }
            }
          },
          "notification_id": {
            "type": "string"
          },
          "object_id": {
            "type": "string",
            "pattern": "^%space_id_pattern%$"
          },
          "object_type": {
            "type": "string",
            "enum": [
              "storagespace"
            ]
          },
          "subject": {
            "type": "string",
            "enum": [
              "Removed from Space"
            ]
          },
          "subjectRich": {
            "type": "string",
            "enum": [
              "Removed from Space"
            ]
          },
          "user": {
            "type": "string",
            "enum": [
              "Alice"
            ]
          }
        }
      }
      """


  Scenario: get a notification of space disabled
    Given user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And user "Alice" has disabled a space "notification checking"
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And user "Brian" should have "2" notifications
    And the JSON response should contain a notification message with the subject "Space disabled" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "app",
          "datetime",
          "message",
          "messageRich",
          "messageRichParameters",
          "notification_id",
          "object_id",
          "object_type",
          "subject",
          "subjectRich",
          "user"
        ],
        "properties": {
          "app": {
            "type": "string",
            "enum": [
              "userlog"
            ]
          },
          "message": {
            "type": "string",
            "enum": [
              "Alice Hansen disabled Space notification checking"
            ]
          },
          "messageRich": {
            "type": "string",
            "enum": [
              "{user} disabled Space {space}"
            ]
          },
          "messageRichParameters": {
            "type": "object",
            "required": [
              "space",
              "user"
            ],
            "properties": {
              "space": {
                "type": "object",
                "required": [
                  "id",
                  "name"
                ],
                "properties": {
                  "id": {
                    "type": "string",
                    "pattern": "^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}\\$[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}![a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$"
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "notification checking"
                    ]
                  }
                }
              },
              "user": {
                "type": "object",
                "required": [
                  "displayname",
                  "id",
                  "name"
                ],
                "properties": {
                  "displayname": {
                    "type": "string",
                    "enum": [
                      "Alice Hansen"
                    ]
                  },
                  "id": {
                    "type": "string",
                    "enim": [
                      "%user_id%"
                    ]
                  },
                  "name": {
                    "type": "string",
                    "enum": [
                      "Alice"
                    ]
                  }
                }
              }
            }
          },
          "notification_id": {
            "type": "string"
          },
          "object_id": {
            "type": "string",
            "pattern": "^%space_id_pattern%$"
          },
          "object_type": {
            "type": "string",
            "enum": [
              "storagespace"
            ]
          },
          "subject": {
            "type": "string",
            "enum": [
              "Space disabled"
            ]
          },
          "subjectRich": {
            "type": "string",
            "enum": [
              "Space disabled"
            ]
          },
          "user": {
            "type": "string",
            "enum": [
              "Alice"
            ]
          }
        }
      }
      """


  Scenario Outline: get a notification about a space share in various languages
    Given user "Brian" has switched the system language to "<language>"
    And user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the JSON response should contain a notification message with the subject "<subject>" and the message-details should match
      """
      {
        "type": "object",
        "required": [
          "message"
        ],
        "properties": {
          "message": {
            "type": "string",
            "enum": [
              "<message>"
            ]
          }
        }
      }
      """
    Examples:
      | language | subject           | message                                                         |
      | de       | Space freigegeben | Alice Hansen hat Sie zu Space notification checking hinzugefügt |
      | es       | Space compartido  | Alice Hansen te añadió al Space notification checking           |


  Scenario: all notification related to space get deleted when the sharer deletes that resource
    Given user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And user "Alice" has unshared a space "notification checking" shared with "Brian"
    And user "Alice" has disabled a space "notification checking"
    And user "Alice" has deleted a space "notification checking"
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And the notifications should be empty


  Scenario: user doesn't get any notification after being removed from space
    Given user "Alice" has shared a space "notification checking" with settings:
      | shareWith | Brian  |
      | role      | editor |
    And user "Alice" has unshared a space "notification checking" shared with "Brian"
    And user "Alice" has disabled a space "notification checking"
    When user "Brian" lists all notifications
    Then the HTTP status code should be "200"
    And user "Brian" should have "2" notifications
