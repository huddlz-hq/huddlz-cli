Feature: Profile search defaults
  Scenario: Search using my profile home location
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I search using my saved profile
    Then the command succeeds
    And the search uses my current profile location

  Scenario: Explicitly search everywhere despite a profile default
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I search everywhere despite my profile
    Then the command succeeds
    And the profile is bypassed without a geographic restriction

  Scenario Outline: Search clearly without a usable default
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given my profile location is <state>
    When I search using my saved profile
    Then the command succeeds
    And the profile lookup is followed by an unrestricted search

    Examples:
      | state      |
      | unset      |
      | incomplete |

  Scenario: A profile failure is not an absent preference
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given my profile location is unavailable
    When I search using my saved profile
    Then the command fails with "Could not read search defaults: HTTP 503"

  Scenario: Signed-in discovery uses the saved session
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I search everywhere despite my profile
    Then the command succeeds
    And discovery sends my saved session

  Scenario: Inspect authorized virtual and hosting details
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I inspect a huddl using my saved session
    Then the command succeeds
    And I see the disclosed virtual link and hosting group

  Scenario: Rejected discovery credentials do not fall back to anonymous access
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given discovery rejects my saved session
    When I search everywhere despite my profile
    Then the command fails with "Search failed: HTTP 401"
    And discovery sends my saved session
