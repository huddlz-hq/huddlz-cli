Feature: Search near a chosen location
  Scenario Outline: Use explicit coordinates for this search only
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | --lat    |
      | <lat>    |
      | --lng    |
      | <lng>    |
      | --radius |
      | <radius> |
    Then the API receives these search filters:
      | date_filter      | upcoming |
      | search_time_zone | Etc/UTC  |
      | sort             | starts_at |
      | page[limit]      | 20       |
      | page[offset]     | 0        |
      | search_latitude  | <lat>    |
      | search_longitude | <lng>    |
      | distance_miles   | <radius> |
    And the output describes a search near "<lat>", "<lng>" within "<radius>" miles
    And I see the matching huddlz in a readable table
    And no profile update is requested
    And the command succeeds

    Examples:
      | lat     | lng      | radius |
      | 40.7128 | -74.006  | 25     |
      | 0       | 0        | 5      |
      | -90     | 180      | 100    |

  Scenario: Keep the chosen location when continuing a search
    Given the public API has two pages of matching huddlz
    When I search the first page for "board games" with nearby filters
    Then I see the first matching huddl and a next-page command
    And the output describes a search near "0", "0" within "25" miles
    When I run the suggested next-page command
    Then I see the second matching huddl
    And both requests preserve the query and filters
    And I am told there are no more results

  Scenario Outline: Reject incomplete location input
    Given the public API has upcoming board game huddlz
    When I search with "<option>" set to "<value>"
    Then the command rejects "<option>" without searching

    Examples:
      | option   | value |
      | --lat    | 40    |
      | --lng    | -74   |
      | --radius | 25    |

  Scenario Outline: Reject invalid coordinates and radii
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | --lat    |
      | <lat>    |
      | --lng    |
      | <lng>    |
      | --radius |
      | <radius> |
    Then the command rejects "<option>" without searching

    Examples:
      | lat  | lng  | radius | option   |
      | 91   | 0    | 25     | --lat    |
      | -91  | 0    | 25     | --lat    |
      | NaN  | 0    | 25     | --lat    |
      | +Inf | 0    | 25     | --lat    |
      | 0    | 181  | 25     | --lng    |
      | 0    | -181 | 25     | --lng    |
      | 0    | NaN  | 25     | --lng    |
      | 0    | -Inf | 25     | --lng    |
      | 0    | 0    | 4      | --radius |
      | 0    | 0    | 101    | --radius |
      | 0    | 0    | 5.5    | --radius |

  Scenario: Reject contradictory geographic scopes
    Given the public API has upcoming board game huddlz
    When I search with these arguments:
      | --anywhere |
      | --lat      |
      | 0          |
      | --lng      |
      | 0          |
    Then the command rejects "--anywhere" without searching
