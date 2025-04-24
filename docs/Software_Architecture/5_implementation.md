# 🔧 Implement & 🐳 deployment

## Strategies and Processes

This subsection outlines the specific methods and steps for integrating and deploying the credimi solution, emphasizing its evolution through a user-centric approach. The architecture design was refined significantly based on feedback gathered through detailed questionnaires and user interviews. This feedback highlighted the need for robust handling of long-running tasks, seamless scheduling, and fault tolerance, leading to the introduction of Temporal.io into the architecture.

The inclusion of Temporal.io enables credimi to deliver enhanced capabilities, such as reliable state management, advanced scheduling, exponential retry mechanisms, and support for human-in-the-loop workflows. These benefits directly address user needs for scalability, resilience, and usability, making the platform more adaptable and effective in managing compliance workflows and reporting processes.

Through this iterative design process, credimi now leverages Temporal.io alongside a modular API and robust monitoring stack, ensuring it provides an intuitive, reliable, and user-focused conformance tool for verifying and managing compliance with European digital identity standards.

### Step-by-Step Deployment

The deployment process for credimi is streamlined through a fully containerized infrastructure managed by Docker. Each component, from the backend services to monitoring tools, is packaged into containers, ensuring consistency across development, testing, and production environments.

1. Containerized Services: All core components (e.g., API gateway, Temporal.io, PostgreSQL, Grafana, Prometheus) are deployed as Docker containers for seamless integration and scalability.
2. Mock Services for Testing:
    - Mailhog simulates mailing functionality, ensuring email workflows can be tested without external dependencies.
    - A third-party simulator acts as a mock for external compliance tools, introducing intentional faults to stress-test the system.
    - Additional external API mocks ensure all integrations are validated before real-world connections are established.
3. Pre-Staging Validation: Before progressing to the staging environment, these mocks provide a controlled setup to test all workflows and API interactions comprehensively.

This containerized and mock-driven approach reduces dependency risks, accelerates testing, and ensures a reliable transition to staging and production environments.

### Strategy Plan: From Development to Production

#### 1. Development Phase

- **Containerized Architecture** ✅:

    - Ensure all services are fully containerized using Docker for consistent environments.
    - Validate the configuration of core components such as Temporal.io, PostgreSQL, API gateway, and monitoring tools (Prometheus, Grafana).

- **Mock Services** ✅:

    - Deploy **Mailhog** for testing email workflows.
    - Use the **third-party simulator** to test compliance integrations, including error scenarios.
    - Set up **external API mocks** to replicate third-party dependencies for controlled testing.

- **Workflow Design** 👷:

    - Develop workflows using Temporal.io for compliance checks, reporting, and scheduling.
    - Test workflows locally and validate state management, retries, and external signal handling.
    - Integrate key components:
    - Establish seamless connectivity between the Compliance Engine and external conformance tools via RESTful APIs.

- **Platform Interoperability** 👷:
    - Implement multi-standard support (eIDAS 2.0, EUDI Wallet, OpenID) to enable broad adoption across European ecosystems.
    - Ensure backward compatibility with DIDroom's existing identity wallet infrastructure to extend TrustChain functionality.

#### 2. Pre-Staging Testing

- **Integration Testing**:

    - Combine all services in a local environment with mocks to ensure end-to-end functionality.
    - Validate all API endpoints, ensuring responses adhere to the Google JSON Style Guide.

- **User Experience Validation**:
    - Ensure the dashboard and UI are user-friendly, with clear navigation for developers and end-users.
    - Gather feedback from stakeholders during this phase for iterative refinements.

#### 3. Staging Environment

- **Isolated Deployment**:

    - Set up a staging environment mirroring production, with isolated database instances and Temporal namespaces.
    - Replace mock services with real integrations where feasible.

- **Data Management**:

    - Load realistic test data into the staging environment to simulate production scenarios.
    - Verify compliance workflows and reporting outputs with actual data.

- **Monitoring and Observability**:
    - Configure Grafana dashboards and Prometheus alerts to monitor system health and performance.
    - Track workflow execution metrics via Temporal UI.

#### 4. Pre-Production Validation

- **Security Audits**:

    - Perform a security audit of the system, focusing on API endpoints, data handling, and user access controls.

- **Stakeholder Approval**:
    - Present the staging environment to stakeholders and finalize any adjustments based on their feedback.

#### 5. Production Rollout

- **Zero-Downtime Deployment**:

    - Use rolling updates to deploy the system into production, ensuring no downtime.
    - Monitor system health during deployment to address any issues promptly.

- **Data Migration**:

    - If necessary, migrate existing data into the production database while ensuring integrity.

- **Post-Deployment Testing**:
    - Validate all workflows, APIs, and integrations in the live environment.
    - Ensure Temporal workflows are functioning as expected with real-world usage.

#### 6. Maintenance and Iteration

- **Continuous Monitoring**:

    - Leverage Prometheus and Grafana for ongoing monitoring and alerting.
    - Use Temporal's debugging tools to inspect and resolve any workflow issues.

- **User Feedback Loop**:
    - Gather feedback from real users and stakeholders to refine features and improve the system.

### Technical Components

The following components are critical to the credimi platform within the TrustChain ecosystem:

- **Compliance Engine**: Core module for conducting compliance checks and scoring.
- **API Gateway**: Facilitates secure access to all platform features and supports plugin submissions.
- **Reporting Service**: Generates comprehensive reports with detailed conformance scores.
- **Dashboard and Comparison Tools**: Provides a user-friendly interface for managing services and viewing public compliance data.
- **Plugin Management System**: Allows seamless integration of third-party conformance tools.

@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Component.puml
!define I https://raw.githubusercontent.com/tupadr3/plantuml-icon-font-sprites/master/govicons
!define FA6 https://raw.githubusercontent.com/tupadr3/plantuml-icon-font-sprites/master/font-awesome-4
!include I/user_politician.puml
!include I/user_suit.puml
!include I/ribbon.puml
!include I/presenter.puml

title Container Diagram for credimi

Person(enduser, "End User")
Person(dev, "Developer")
Person(sp, "Service Provider")
Person(gov, "Government IT Manager", $sprite="user_suit")
Person(eu, "EU Official", $sprite="user_politician")
Person(cto, "CTO")
Person(researcher, "Researcher", $sprite="presenter")
Person(sb, "Standardization Body", $sprite="ribbon")

System_Boundary(credimi, "credimi") {
Container(dashboard, "Dashboard", "TypeScript/Svelte", "User interface for managing services and viewing results.")
Container(comparison_tool, "Marketplace/Comparison Tool", "TypeScript/Svelte", "Tool for comparing credential services.")
Container_Boundary(api_gateway, "API Gateway (golang)") {
Component(cli, "CLI", "Command Line Interface", "For direct interaction with the system.")
Component(ci_cd, "CI/CD", "Continuous Integration/Continuous Deployment", "For automated compliance validation.")
Component(api, "API", "REST API", "For programmatic interactions.")
}
Container(compliance_engine, "Compliance Engine", "go, Temporal.io", "Performs compliance checks, integrates plugins, and manages workflows.")
Container(reporting_service, "Reporting Service", "PDF/JSON", "Generates and exports compliance reports.")
}
Container(third, "Third party conformance tools", "Different systems", "Allow to run already in place checks")

Rel(dev, api_gateway, "Submits and checks compliance")
Rel(enduser, comparison_tool, "Browses verified issuers and services")
Rel(sp, dashboard, "Manages and publishes results")
Rel(cto, dashboard, "Schedules and monitors compliance workflows")

Lay_U(ci_cd, api)
Lay_U(api, cli)

Rel(dashboard, api_gateway, "Schedules periodic and ad-hoc checks")
Rel(comparison_tool, api_gateway, "Fetches compliance data for visualization")

Rel_D(api_gateway, compliance_engine, "Processes compliance checks and workflows")
Rel_R(compliance_engine, third, "run check on external tools")

Rel_U(reporting_service, compliance_engine, "Generates reports from stored compliance data")
Rel_U(sb, reporting_service, "Evaluates standards alignment")
Rel_U(gov, reporting_service, "Generates compliance reports")
Rel_U(eu, reporting_service, "Generates compliance reports")
Rel_U(researcher, reporting_service, "Accesses compliance data")
@enduml

The Compliance Engine is composed in detail of the following services/containers

#### ⚙️ credimi-backend (API Gateway)

credimi-credimi is the core service of the project, responsible for handling the primary application logic. It orchestrates all internal processes and ensures that the compliance engine and related functionalities operate cohesively.
This is based on go, specially on pocketbase.io for the API/REST as a framework and Cobra for the CLI.

::: tip RESOURCES
[credimi GitHub](https://github.com/ForkbombEu/credimi)

[Pocketbase](https://github.com/pocketbase/pocketbase)

[Cobra](https://github.com/spf13/cobra)
:::

---

#### 🔄 credimi-temporal

credimi-temporal is the Temporal Workflow service used for managing distributed workflows and task orchestration. It ensures reliability and scalability in handling complex operations by managing dependencies, retries, and task executions.

::: tip RESOURCES
[Documentation](https://docs.temporal.io/)

[GitHub](https://github.com/temporalio/temporal)
:::

---

#### 📧 credimi-mailhog

credimi-mailhog is a testing tool designed to intercept and inspect emails sent by the application. It allows developers to validate email functionalities during the development and testing phases without sending actual emails. This ensures that all email-related operations, such as notifications or verification links, work as intended.

::: tip RESOURCES
[GitHub](https://github.com/mailhog/MailHog)
:::

---

#### 📊 credimi-grafana

credimi-grafana is a powerful visualization and monitoring tool integrated into the project to display metrics and logs from the system. It helps teams gain real-time insights into system performance, track key metrics, and debug issues through customizable dashboards.

::: tip RESOURCES
[Documentation](https://grafana.com/docs/)

[GitHub](https://github.com/grafana/grafana)
:::

---

#### 🛠️ credimi-temporal-admin-tools

credimi-temporal-admin-tools provides administrative tools for managing the Temporal Workflow service. These tools allow developers to configure, monitor, and debug distributed workflows and ensure smooth task orchestration across the system.

::: tip RESOURCES
[Documentation](https://docs.temporal.io/)

[GitHub](https://github.com/temporalio/temporal)
:::

---

#### 👀 credimi-temporal-ui

credimi-temporal-ui is a user interface for managing and observing workflows in the Temporal service. It provides a visual representation of workflows, their statuses, and debugging tools, making workflow management more intuitive for administrators and developers.

::: tip RESOURCES
[Documentation](https://docs.temporal.io/ui/)

[GitHub](https://github.com/temporalio/ui-server)
:::

---

#### 📈 credimi-prometheus

credimi-prometheus is a metrics collection and monitoring tool integrated into the project. It scrapes, stores, and visualizes application metrics, enabling teams to track performance, detect anomalies, and optimize system behavior proactively.

::: tip RESOURCES
[Documentation](https://prometheus.io/docs/)

[GitHub](https://github.com/prometheus/prometheus)
:::

---

#### 🗄️ credimi-postgresql

credimi-postgresql is the PostgreSQL database instance used as the backend for data storage. It serves as the core storage layer for compliance data, user information, and reports, ensuring data integrity and efficient query handling.

::: tip RESOURCES
[Documentation](https://www.postgresql.org/docs/)

[GitHub](https://github.com/postgres/postgres)
:::

---

#### 🌐 credimi-thirdparty

credimi-thirdparty acts as a placeholder service for external integrations or third-party API interactions. This module facilitates seamless connectivity between the core system and various external services, ensuring interoperability and extensibility for future integrations.

::: tip RESOURCES
[GitHub](https://github.com/ForkbombEu/credimi)
:::

---

@startuml
!include <C4/C4_Component>

!define DEVICONS https://raw.githubusercontent.com/tupadr3/plantuml-icon-font-sprites/master/devicons
!define FONTAWESOME https://raw.githubusercontent.com/tupadr3/plantuml-icon-font-sprites/master/font-awesome-5
!include DEVICONS/go.puml
!include FONTAWESOME/users.puml

title Component Diagram for Compliance Engine

Container_Boundary(compliance_engine, "Compliance Engine", "go, Temporal.io", "Performs compliance checks, integrates plugins, and manages workflows.") {
Component(database, "Database", "PostgreSQL", "Stores compliance data, user data, and reports.")
Component(plugin_system, "Plugin Management", "Go / Docker", "Handles third-party plugin integration and execution.")
Component(prometheus, "Prometheus", "Monitoring", "Collects metrics from all services.")
Component(grafana, "Grafana", "Visualization", "Provides real-time dashboards and alerts.")
Component(temporal_ui, "Debug dashboard", "", "Allows to playback error visually debug workflows and repeat processes")
Component(temporal, "Temporal", "Workflow manager", "Handles long running processes and allow visible workflow debugs")
Component(backend, "Backend", "Pocketbase/temporal", "The business login and orchestrator of the engine")
}

Rel_U(backend, temporal, "Manages the conformance check workflows")
Rel(backend, plugin_system, "Loads and executes third-party plugins")
Rel_L(plugin_system, database, "Stores plugin configurations")
Rel_R(prometheus, backend, "Collects metrics from workflows")
Rel(prometheus, database, "Tracks database metrics")
Rel(grafana, prometheus, "Visualizes metrics and generates alerts")
Rel_U(temporal, temporal_ui, "try and repeat")

@enduml

---

## API Specification Evolution (Final)

Based on the user feedback we collected the following points

| **Key Points**                                                                        | **API Impact**                                                                 |
| ------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| Enhance plugin management systems to allow seamless integration of third-party tools. | Ensure plugin-related endpoints are modular and allow easy extensions.         |
| Emphasize status transparency in long-running tasks (e.g., compliance checks).        | Refine endpoints related to scheduling and task monitoring.                    |
| Improve response clarity for some endpoints.                                          | Provide a clear distinction between error types and their resolutions.         |
| Add sandbox environments for testing integrations.                                    | Introduce sandbox APIs to test integrations.                                   |
| Support batch operations for compliance checks.                                       | Add batch processing capabilities for compliance and reporting.                |
| Include multi-standard compliance in single API calls.                                | Enhance reporting APIs to support multiple standards per request.              |
| Provide detailed conformance scores in compliance reports.                            | Refine the compliance score module to offer more granular insights.            |
| Support team-based organization management.                                           | Extend organization-related endpoints to include team management.              |
| Enable feature-rich dashboard API for UI integrations.                                | Optimize dashboard endpoints to support complex UI widgets.                    |
| Include historical data in compliance results for trend analysis.                     | Update APIs to include historical compliance data.                             |
| Address privacy concerns and allow control over report visibility.                    | Add advanced privacy controls for reports and services.                        |
| Ensure interoperability with legacy systems.                                          | Ensure all APIs are backward-compatible with legacy standards.                 |
| Add claim ownership APIs for organization management.                                 | Extend endpoints to allow users to claim ownership of organizations.           |
| Align error responses with the Google JSON Style Guide.                               | Revamp error handling to provide actionable recommendations in error messages. |

These can be summarized into the following:

### Changes and Justifications

Based on feedback, the following changes were made:

- **Added Plugin Management Endpoints**: Supports third-party conformance tool integrations, addressing developer requests for extensibility.
- **Enhanced Reporting APIs**: Included granular compliance scoring and batch processing capabilities.
- **Improved Privacy Controls**: Introduced endpoints for managing report visibility, ensuring GDPR compliance.
- **Introduced Batch Operations**: Simplified compliance checks for multiple services at once, increasing efficiency.
- **Error reporting**: Ensure error handling aligns with the Google JSON Style Guide.
- **Sandbox and testing environements**: Add sandbox and testing environments of services like a fake/mock credential isuer or relying party to allow devs to test while developing their products.

### Consistency

Core endpoints for compliance checks, organization management, and scheduling remain unchanged as they continue to meet project objectives and align with TrustChain’s vision. Also some endpoints like user management just changed the address to reflect directly the utilities provided by PocketBase.io that is the backend framework in go choosen that consists of an embedded database (SQLite) with realtime subscriptions, built-in auth management, convenient dashboard UI and simple REST-ish API.

### Design Evolution

User feedback influenced iterative refinements:

1.  Functionality: Adds critical features like plugin management and batch operations.
2.  Usability: Ensures clearer error handling and privacy controls.
3.  Interoperability: Addresses legacy system support and multi-standard compliance. Added support for multi-standard conformance in single API calls, increasing system versatility.
4.  Scalability: Prepares the API for future growth by supporting advanced use cases.

---

## Platform Identification and Integration

### Selected Platform: DIDroom Integration

DIDroom serves as a foundational platform for credimi, providing an open-source identity solution that aligns with TrustChain’s goals of interoperability and decentralized identity management.

### Evaluation

- **Interoperability**: Supports integration with DIDroom’s existing W3C-compliant identity wallets, facilitating cross-standard compatibility.
- **Scalability**: DIDroom’s modular architecture ensures the ability to handle growing transaction volumes and new compliance demands.
- **Security**: Integrates robust cryptographic protocols to maintain data integrity and secure operations.
- **Regulatory Compliance**: Fully adheres to GDPR and European standards for privacy and governance.
- **Customization**: Allows the extension of existing DIDroom libraries to support TrustChain-specific use cases.
- **Efficiency**: Optimized workflows reduce redundancy, leveraging DIDroom’s reusable identity management components.

### Integration with Existing Tools

credimi leverages components and tools from DIDroom to strengthen the TrustChain ecosystem:

- **Identity Wallets**: DIDroom identity wallets provide a secure basis for credential management.
- **Libraries**: Extends decentralized identity libraries already developed under DIDroom.
- **External Tools**: Integrates with external conformance and testing tools with the power of our workflow management.

For now we have planned to use as external conformance platforms, some based on [GITB](https://www.itb.ec.europa.eu/docs/tdl/latest/index.html) that is choosen
by the european commision as the standard testbed platform for all different projects.
In particular:

- EWC wallet conformance tool https://github.com/EWC-consortium/ewc-wallet-conformance-backend
- OpenID Conformance suite to validate the openid4vp and in general the openid4vci standard https://gitlab.com/openid/conformance-suite/
- The test tools suggested by the Wallet Interoperability Special Interest Group (WISIG) of the OpenWallet Foundation https://tac.openwallet.foundation/SIGs/wallet-interoperability/
- The EBSI conformance testing platform https://hub.ebsi.eu/wallet-conformance
