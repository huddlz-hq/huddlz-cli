Feature: RSVP to a huddl
  Scenario: RSVP when a huddl has space available
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I RSVP to a huddl with space available
    Then the command succeeds
    And attendance is confirmed by the backend

  Scenario Outline: Backend RSVP rejections remain failures
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the RSVP endpoint returns HTTP <status>
    When I RSVP to a huddl with space available
    Then the command fails with "RSVP request failed; attendance was not confirmed: HTTP <status>"
    And only one RSVP attempt was made

    Examples:
      | status |
      | 403    |
      | 404    |
      | 422    |
      | 503    |

  Scenario: Expired credentials cannot RSVP
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the RSVP endpoint returns HTTP 401
    When I RSVP to a huddl with space available
    Then authentication is required without prompting or exposing credentials
    And only one RSVP attempt was made

  Scenario: A successful submission does not imply confirmed attendance
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the server cannot confirm my attendance
    When I RSVP to a huddl with space available
    Then the command fails with "RSVP was submitted, but the server did not confirm attendance"
    And the RSVP was submitted only once before checking attendance

  Scenario: RSVP requires a saved session
    Given the authentication API accepts my credentials
    When I RSVP to a huddl with space available
    Then the command fails with "not logged in to this server"
    And no authenticated request follows sign-out
