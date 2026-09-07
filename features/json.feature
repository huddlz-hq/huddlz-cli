Feature: Structured search output
  Scenario: Preserve results and search context as JSON
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | --json |
      | --lat  |
      | 0      |
      | --lng  |
      | 0      |
      | board games |
    Then the command succeeds
    And JSON preserves the search results and context

  Scenario: Empty results are a JSON array
    Given the public API has no matching huddlz
    When I search with these arguments:
      | --json |
    Then the command succeeds
    And JSON contains an empty results array

  Scenario: Failures do not corrupt JSON output
    Given the public API fails with "HTTP error"
    When I search with these arguments:
      | --json |
    Then the command fails with "Search failed: HTTP 503"

  Scenario Outline: Preserve JSON pagination information
    Given the JSON search API provides pagination "<next>"
    When I search with these arguments:
      | --json |
      | --type |
      | virtual |
      | --limit |
      | 1 |
    Then the command succeeds
    And JSON reports pagination "<next>"

    Examples:
      | next    |
      | next    |
      | end     |
      | unknown |

  Scenario: Invalid pagination does not produce partial JSON
    Given the JSON search API provides pagination "invalid"
    When I search with these arguments:
      | --json |
    Then the command fails with "Search failed:"
