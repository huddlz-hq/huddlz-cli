Feature: Review my RSVPs
  Scenario: Distinguish confirmed attendance from waitlist entries
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I list my RSVPs
    Then the command succeeds
    And my RSVP list identifies confirmed and waitlisted huddlz

  Scenario: Follow a waitlisted page while preserving structured output
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given my RSVP list has another waitlisted page
    When I list my RSVPs as JSON
    Then the command succeeds
    And JSON distinguishes membership and provides a waitlisted continuation
    When I run the RSVP continuation command
    Then the command succeeds
    And the continuation lists only the next waitlisted page

  Scenario: Empty membership is an explicit empty result
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given my RSVP lists are empty
    When I list my RSVPs as JSON
    Then the command succeeds
    And JSON contains empty membership groups

  Scenario: A failed waitlist lookup does not produce a partial list
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given the waitlisted lookup fails
    When I list my RSVPs
    Then the command fails with "Could not list RSVPs: HTTP 503"

  Scenario: A saved session is required to list RSVPs
    Given the authentication API accepts my credentials
    When I list my RSVPs
    Then the command fails with "not logged in to this server"
