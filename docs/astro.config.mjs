import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import credimiLogo from "./src/content/docs/images/logo/credimi_logo-transp_emblem.png";
import remarkPlantuml from "./remark-plantuml.mjs";

export default defineConfig({
    site: "https://docs.credimi.io",
    markdown: {
        remarkPlugins: [remarkPlantuml],
    },
    integrations: [
        starlight({
            title: "Credimi Docs",
            logo: { src: credimiLogo },
            sidebar: [
                {
                    label: "Manual",
                    items: [
                        { label: "Overview", link: "/manual/" },
                        {
                            label: "Hub",
                            items: [
                                {
                                    label: "Overview",
                                    link: "/manual/hub/",
                                },
                                {
                                    label: "Explore Components and Solutions",
                                    link: "/manual/hub/explore-components-and-solutions/",
                                },
                                {
                                    label: "Try Issuance and Verification Services",
                                    link: "/manual/hub/try-issuance-and-verification-services/",
                                },
                            ],
                        },
                        {
                            label: "Conformance & Interop",
                            items: [
                                {
                                    label: "Overview",
                                    link: "/manual/conformance/",
                                },
                                {
                                    label: "Run Conformance Checks",
                                    link: "/manual/conformance/run-conformance-checks/",
                                },
                                {
                                    label: "Try Interop Flows",
                                    link: "/manual/conformance/try-interop-flows/",
                                },
                            ],
                        },
                        {
                            label: "Publish to Hub",
                            items: [
                                {
                                    label: "Overview",
                                    link: "/manual/publish-to-hub/",
                                },
                                {
                                    label: "Create an Account and Organization",
                                    link: "/manual/publish-to-hub/create-account-and-organization/",
                                },
                                {
                                    label: "List Wallets, Issuers and Verifiers",
                                    link: "/manual/publish-to-hub/list-wallets-issuers-and-verifiers/",
                                },
                                {
                                    label: "Integrate Issuers and Verifiers with StepCI",
                                    link: "/manual/publish-to-hub/integrate-with-stepci/",
                                },
                                {
                                    label: "StepCI and Maestro scripting examples",
                                    link: "/manual/yaml-examples/",
                                },
                            ],
                        },
                        {
                            label: "Testing Automation",
                            items: [
                                {
                                    label: "Overview",
                                    link: "/manual/testing-automation/",
                                },
                                {
                                    label: "Create Maestro Actions",
                                    link: "/manual/testing-automation/create-maestro-actions/",
                                },
                                {
                                    label: "Build and Run Pipelines",
                                    link: "/manual/testing-automation/build-and-run-pipelines/",
                                },
                                {
                                    label: "Inspect Pipeline Execution",
                                    link: "/manual/testing-automation/inspect-pipeline-execution/",
                                },
                                {
                                    label: "StepCI and Maestro scripting examples",
                                    link: "/manual/yaml-examples/",
                                },
								                                {
                                    label: "Pipeline utils",
                                    link: "/manual/testing-automation/pipeline-utils/",
                                },
                            ],
                        },
                        {
                            label: "Credimi Runner",
                            items: [
                                {
                                    label: "Overview",
                                    link: "/manual/credimi-runner/",
                                },
                                {
                                    label: "Setup your runner",
                                    link: "/manual/credimi-runner/credimi-runner-setup/",
                                },
                                {
                                    label: "Step-by-step CLI setup",
                                    link: "/manual/credimi-runner/credimi-runner-setup-explained/",
                                },
                                {
                                    label: "API Keys",
                                    link: "/manual/credimi-runner/api-keys/",
                                },
                            ],
                        },
                        {
                            label: "CI/CD Integration",
                            items: [
                                { label: "Overview", link: "/manual/ci-cd/" },
                            ],
                        },
                    ],
                },
                {
                    label: "Legal",
                    autogenerate: { directory: "legal" },
                },
                {
                    label: "API Reference",
                    link: "/API/",
                },
            ],
        }),
    ],
});
