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
