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

  Scenario: Explain expired credentials on an authenticated operation
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given my saved credentials have expired
    When I check authentication status in a new process
    Then authentication is required without prompting or exposing credentials
    And only the rejected account verification was attempted

  Scenario Outline: Account verification failures do not imply expired credentials
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given account verification returns HTTP <status>
    When I check authentication status in a new process
    Then the command fails with "Account verification failed: HTTP <status>"
    And only a private session token is saved

    Examples:
      | status |
      | 403    |
      | 503    |

  Scenario: Stop using a saved session after sign-out
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    When I sign out
    Then the command succeeds
    And sign-out revokes the saved session without exposing it
    When I check authentication status in a new process
    Then the command fails with "not logged in to this server"
    And no authenticated request follows sign-out

  Scenario Outline: Sign-out stops local use when server revocation fails
    Given the authentication API accepts my credentials
    When I log in with my password on stdin
    Then the command succeeds
    Given server revocation fails with "<failure>"
    When I sign out
    Then local sign-out reports unconfirmed server revocation
    When I check authentication status in a new process
    Then the command fails with "not logged in to this server"
    And no authenticated request follows sign-out

    Examples:
      | failure      |
      | expired      |
      | unavailable  |
      | disconnected |

  Scenario: Sign-out with no saved session is harmless
    Given the authentication API accepts my credentials
    When I sign out
    Then the command succeeds
    And I am already signed out without a server request
