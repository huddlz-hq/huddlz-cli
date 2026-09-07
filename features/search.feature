Feature: Discover public huddlz
  Scenario: Combine date and type search filters
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | --date           |
      | this_week        |
      | --type           |
      | in_person        |
      | --time-zone      |
      | America/New_York |
    Then the API receives these search filters:
      | date_filter      | this_week        |
      | event_type       | in_person        |
      | search_time_zone | America/New_York |
      | sort             | starts_at        |
      | page[limit]      | 20               |
    And the output describes filters "this_week", "in_person", and "America/New_York"
    And I see the matching huddlz in a readable table
    And the command succeeds

  Scenario Outline: Reject invalid search filters before contacting the API
    Given the public API has upcoming board game huddlz
    When I search with "<option>" set to "<value>"
    Then the command rejects "<option>" without searching

    Examples:
      | option      | value         |
      | --date      | tomorrow      |
      | --date      |               |
      | --type      | meeting       |
      | --type      |               |
      | --time-zone | Mars/Olympus  |
      | --time-zone | Local         |
      | --time-zone |               |
      | --time-zone | US/Eastern    |

  Scenario Outline: Search by interest with supported filters
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | board games |
      | --date      |
      | <date>      |
      | --type      |
      | <type>      |
      | --time-zone |
      | <zone>      |
      | --anywhere  |
    Then the API receives these search filters:
      | query            | board games     |
      | date_filter      | <date>          |
      | event_type       | <type>          |
      | search_time_zone | <resolved_zone> |
      | sort             | starts_at       |
      | page[limit]      | 20              |
    And the output describes filters "<date>", "<type>", and "<resolved_zone>"
    And I see the matching huddlz in a readable table
    And the command succeeds

    Examples:
      | date       | type      | zone             | resolved_zone    |
      | upcoming   | in_person | Etc/UTC          | Etc/UTC          |
      | this_week  | virtual   | America/New_York | America/New_York |
      | this_month | hybrid    | Etc/UTC          | Etc/UTC          |
      | past       | virtual   | Europe/London    | Europe/London    |
      | all        | in_person | UTC              | Etc/UTC          |

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
