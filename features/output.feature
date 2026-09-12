Feature: Consistent output
  Scenario Outline: Request structured command results
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given I am attending for a huddl
    When I request JSON from <command>
    Then the command succeeds
    And the command emits a structured result without credentials

    Examples:
      | command       |
      | auth login    |
      | auth status   |
      | auth logout   |
      | show          |
      | rsvp          |
      | rsvp waitlist |
      | rsvp cancel   |
