Feature: Discover public huddlz
  Scenario: Search by interest without signing in
    Given the public API has upcoming board game huddlz
    When I search everywhere for "board games"
    Then the API receives an anonymous bounded upcoming search for "board games"
    And I see the matching huddlz in a readable table
    And the command succeeds
