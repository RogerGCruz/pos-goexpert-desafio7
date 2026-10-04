Feature: Weather via Service A
  Scenario: Get weather for valid CEP
    Given the services are available
    When I POST to service A with cep "01001000"
    Then the response status should be 200
    And the response should contain "Sao Paulo"
