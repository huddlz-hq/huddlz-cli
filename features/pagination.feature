Feature: Continue through matching search results
  Scenario Outline: Request the next page with the same search filters
    Given the public API has two pages of matching huddlz
    When I search the first page for "<query>" with filters
    Then I see the first matching huddl and a next-page command
    When I run the suggested next-page command
    Then I see the second matching huddl
    And both requests preserve the query and filters
    And I am told there are no more results

    Examples:
      | query                                      |
      | a friend's board games; $(echo surprise)    |
      | --type                                     |

  Scenario Outline: Reject invalid page controls before searching
    Given the public API has upcoming board game huddlz
    When I search with "<option>" set to "<value>"
    Then the command rejects "<option>" without searching

    Examples:
      | option   | value |
      | --limit  | 0     |
      | --limit  | -1    |
      | --limit  | 101   |
      | --limit  | many  |
      | --offset | -1    |
      | --offset | next  |

  Scenario: Do not guess whether more results exist without metadata
    Given the public API has upcoming board game huddlz
    When I search with "--limit" set to "100"
    Then the pagination status is unavailable
    And the command succeeds

  Scenario: An empty final page ends the search
    Given the public API has an empty final page
    When I browse upcoming huddlz
    Then I see that no huddlz match
    And I am told there are no more results
    And the command succeeds

  Scenario: Reject a page larger than requested
    Given the public API has upcoming board game huddlz
    When I search with "--limit" set to "1"
    Then the command fails with "more results than requested"
    And no successful results are printed
