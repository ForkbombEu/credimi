// SPDX-FileCopyrightText: 2024 The Forkbomb Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { defineConfig } from "vitepress";
import { generateSidebar } from "vitepress-sidebar";
import umlPlugin from "markdown-it-plantuml";

// https://vitepress.dev/reference/site-config
export default defineConfig({
    title: "EUDI-ARF Compliance/Interop",
    description:
        "Master the complexities of SSI identity solutions with Credimi: Your go-to platform for testing, validating, and ensuring compliance in the ever-evolving digital identity ecosystem.",
    base: "/credimi/",

    lastUpdated: true,
    metaChunk: true,
    ignoreDeadLinks: [/^http?:\/\/localhost/, /^https?:\/\/localhost/],

    head: [
        [
            "script",
            {},
            `window.$crisp=[];window.CRISP_WEBSITE_ID="8dd97823-ddac-401e-991a-7498234e4f00";(function(){d=document;s=d.createElement("script");s.src="https://client.crisp.chat/l.js";s.async=1;d.getElementsByTagName("head")[0].appendChild(s);})();`,
        ],
    ],
    themeConfig: {
        // https://vitepress.dev/reference/default-theme-config
        nav: [
            { text: "Home", link: "/" },
            { text: "Manual", link: "/Manual/index.html" },
            {
                text: "Architecture",
                link: "/Software_Architecture/1_start.html",
            },
            { text: "API Reference", target: "_self", link: "/API/index.html" },
        ],

        sidebar: generateSidebar({
            useTitleFromFileHeading: true,
            sortMenusOrderNumericallyFromLink: true,
        }),
        logo: "",
        socialLinks: [
            { icon: "github", link: "https://github.com/forkbombeu/credimi" },
            { icon: "linkedin", link: "https://linkedin.com/company/forkbomb" },
        ],

        footer: {
            message:
                'Released under the <a href="https://github.com/ForkbombEu/credimi?tab=readme-ov-file#-license">AGPLv3 License</a>.',
            copyright:
                'Copyleft 🄯 2024-present <a href="https://forkbomb.solutions">Forkbomb B.V.</a>',
        },
    },
    markdown: {
        config(md) {
            md.use(umlPlugin);
        },
    },
});
