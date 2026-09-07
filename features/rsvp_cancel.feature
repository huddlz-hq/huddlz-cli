Feature: Cancel attendance
  Scenario Outline: Cancel confirmed attendance or leave a waitlist
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given I am <status> for a huddl
    When I cancel my RSVP
    Then the command succeeds
    And the backend confirms I am no longer attending or waitlisted

    Examples:
      | status     |
      | attending  |
      | waitlisted |

  Scenario: A rejected cancellation remains a failure
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given I am attending for a huddl
    And cancellation is rejected
    When I cancel my RSVP
    Then the command fails with "Cancellation request failed; cancellation was not confirmed: HTTP 403"
    And cancellation was attempted only once

  Scenario: Remaining waitlist membership is not reported as cancelled
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given I am waitlisted for a huddl
    And cancellation leaves membership unchanged
    When I cancel my RSVP
    Then the command fails with "Cancellation was submitted, but the server did not confirm removal"
    And cancellation was submitted once and checked twice

  Scenario: A verification outage is reported honestly
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given I am attending for a huddl
    And cancellation verification is unavailable
    When I cancel my RSVP
    Then the command fails with "Cancellation was submitted, but membership removal could not be verified: HTTP 503"
    And cancellation was submitted once and checked twice

  Scenario: Cancellation requires a saved session
    Given the authentication API accepts my credentials
    When I cancel my RSVP
    Then the command fails with "not logged in to this server"
    And no authenticated request follows sign-out
