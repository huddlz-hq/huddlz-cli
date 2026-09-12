Feature: Join a waitlist
  Scenario Outline: Report the backend attendance outcome
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the waitlist API reports <state>
    When I request waitlist membership
    Then the command succeeds
    And the waitlist result reports <state>

    Examples:
      | state      |
      | waitlisted |
      | confirmed  |

  Scenario: An unknown waitlist outcome is not success
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the waitlist API reports none
    When I request waitlist membership
    Then the command fails with "Attendance state was not confirmed by the API"
