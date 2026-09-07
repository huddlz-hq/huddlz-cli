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
