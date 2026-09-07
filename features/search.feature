Feature: Discover public huddlz
  Scenario: Browse upcoming huddlz without signing in
    Given the public API has upcoming board game huddlz
    When I browse upcoming huddlz
    Then the API receives an anonymous bounded upcoming search for ""
    And I see the matching huddlz in a readable table
    And the command succeeds

  Scenario: Nothing matches the search
    Given the public API has no matching huddlz
    When I search everywhere for "board games"
    Then the API receives an anonymous bounded upcoming search for "board games"
    And I see that no huddlz match
    And the command succeeds

  Scenario Outline: API failures are not empty successful searches
    Given the public API fails with "<failure>"
    When I search everywhere for "board games"
    Then the command fails with "<message>"
    And no successful results are printed

    Examples:
      | failure          | message                                      |
      | HTTP error       | Search failed: HTTP 503 Service Unavailable   |
      | disconnected     | Search request failed:                       |
      | invalid JSON     | invalid or oversized JSON:API response        |
      | missing data     | invalid or oversized JSON:API response        |
      | null data        | invalid or oversized JSON:API response        |
      | null resource    | invalid or oversized JSON:API response        |
      | empty resource   | invalid or oversized JSON:API response        |

  Scenario: Search by interest without signing in
    Given the public API has upcoming board game huddlz
    When I search everywhere for "board games"
    Then the API receives an anonymous bounded upcoming search for "board games"
    And I see the matching huddlz in a readable table
    And the command succeeds
