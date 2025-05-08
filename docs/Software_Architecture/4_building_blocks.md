<!--
SPDX-FileCopyrightText: 2024 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# 🏗️ Building blocks

The following system diagrams provide a visual overview of the credimi platform, illustrating its architecture and key components. These diagrams are based on the C4 model, which is a widely-used approach for visualizing software architecture.

While we follow the principles of the C4 model, our implementation is somewhat loose, focusing on clarity and relevance to the specific context of credimi. The diagrams capture different levels of abstraction, from the overall system context down to detailed component interactions within the platform. This structured approach helps in understanding how various parts of the system interact and contribute to its overall functionality.

## System Context diagrams

Shows the interaction between external actors (developers, service providers, governmental bodies, etc.) and the credimi system.

@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml
!define I https://raw.githubusercontent.com/tupadr3/plantuml-icon-font-sprites/master/govicons
!include I/users.puml
!include I/user_politician.puml
!include I/user_suit.puml
!include I/ribbon.puml
!include I/presenter.puml

LAYOUT_WITH_LEGEND()

title System Context Diagram for credimi

Person(dev, "Developer", "Submits credential issuers and runs compliance checks.")
Person(sp, "Service Provider", "Manages and publishes compliance results.")
Person(gov, "Government IT Manager", "Ensures compliance with standards.", $sprite="user_suit")
Person(eu, "EU Official", "Generates compliance reports.", $sprite="user_politician")
Person(cto, "CTO", "Browses and compares credential services.")
Person(enduser, "End User", "Browses verified credential issuers.", $sprite="users")
Person(researcher, "Researcher", "Analyzes compliance data.", $sprite="presenter")
Person(sb, "Standardization Body", "Evaluates standards alignment.", $sprite="ribbon")

System(credimi, "credimi", "Platform for verifying compliance of decentralized identity services.")

Rel(dev, credimi, "Submits and checks compliance")
Rel_U(sp, credimi, "Manages and publishes results")
Rel_U(gov, credimi, "Generates compliance reports")
Rel_R(eu, credimi, "Generates compliance reports")
Rel_R(cto, credimi, "Browses and compares services")
Rel_L(enduser, credimi, "Browses verified issuers")
Rel_L(researcher, credimi, "Analyzes compliance data")
Rel(sb, credimi, "Evaluates standards alignment")
Lay_U(enduser, researcher)
Lay_U(cto, eu)
@enduml

## Container diagrams

Illustrates the main containers within the credimi system (API Gateway, Compliance Engine, Dashboard, etc.) and their interactions.

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

## Component Diagram

Focuses on the internal components of the Compliance Engine, showing how the various modules work together.

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
