Feature: Weather edge cases

  Scenario: Invalid CEP format
    Given the services are available
    When I POST to service A with cep "01001"
    Then the response status should be 422
    And the response should contain "invalid zipcode"

  Scenario: Non-existent CEP returns 404
    Given the services are available
    When I POST to service A with cep "00000000"
    Then the response status should be 404
    And the response should contain "can not find zipcode"
