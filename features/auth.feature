Feature: Authenticate and verify the current account
  Scenario: Log in without exposing credentials and verify the saved session
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    And I see my account without credentials
    When I check authentication status in a new process
    Then the command succeeds
    And I see my account without credentials
    And the server verified my saved session

  Scenario: Save only the session token with private permissions
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    And only a private session token is saved

  Scenario: A saved session belongs to its server
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I check authentication status for another server
    Then the command fails with "not logged in to this server"

  Scenario: Login does not forward credentials through a redirect
    Given the authentication API accepts my credentials
    And the login endpoint redirects to another destination
    When I log in with my password on stdin
    Then the command fails with "Login failed: HTTP 307"
    And no redirected credential request is made

  Scenario: Login rejects plaintext remote servers before sending credentials
    Given the authentication API accepts my credentials
    When I attempt login over remote plaintext HTTP
    Then authentication rejects the insecure server without a request

  Scenario: Report rejected authentication without creating a session
    Given the authentication API accepts my credentials
    When I log in with an incorrect password
    Then the rejected login is reported without credentials
    When I check authentication status in a new process
    Then the command fails with "not logged in to this server"
    And no account verification followed the rejected login

  Scenario: Rejected credentials preserve an existing valid session
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I log in with an incorrect password
    Then the rejected login is reported without credentials
    And only a private session token is saved
    When I check authentication status in a new process
    Then the command succeeds
    And I see my account without credentials
