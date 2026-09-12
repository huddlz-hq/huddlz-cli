Feature: Everyday command use
  Scenario: Bound a slow API request
    Given the authentication API accepts my credentials
    And the API responds slowly
    When I search with a short timeout
    Then the command fails with "Search request failed:"
    And no automatic request retry occurs

  Scenario Outline: Generate shell completion
    Given the authentication API accepts my credentials
    When I request <shell> completion
    Then the command succeeds
    And completion includes CLI commands

    Examples:
      | shell |
      | bash  |
      | zsh   |
      | fish  |

  @unix
  Scenario: Interrupt an in-flight request
    Given the authentication API accepts my credentials
    And the API responds slowly
    When I interrupt a running search
    Then the command exits as interrupted without successful output

  Scenario: Complete only flags accepted by the selected subcommand
    Given the authentication API accepts my credentials
    When I request bash completion
    Then bash completion respects the selected subcommand
