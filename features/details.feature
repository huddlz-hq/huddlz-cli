Feature: Inspect a public huddl
  Scenario: Inspect a huddl found in search
    Given a public huddl has published details
    When I look up the huddl by its identifier
    Then I see its public details and availability limits

  Scenario: Inspect a huddl with undisclosed optional details
    Given a public huddl has published details
    And its optional details are undisclosed
    When I look up the huddl by its identifier
    Then missing details are reported honestly

  Scenario Outline: Report a missing or inaccessible huddl
    Given the huddl lookup returns HTTP <status>
    When I look up the huddl by its identifier
    Then the huddl is reported unavailable without disclosing details
    And no alternate huddl lookup is attempted

    Examples:
      | status |
      | 401    |
      | 403    |
      | 404    |

  Scenario: A server outage is not reported as a missing huddl
    Given the huddl lookup returns HTTP 503
    When I look up the huddl by its identifier
    Then the command fails with "Huddl lookup failed: HTTP 503 Service Unavailable"
    And no alternate huddl lookup is attempted
