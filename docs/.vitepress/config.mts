import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "GripMock",
  titleTemplate: 'gRPC Mock Server Documentation',
  description: "GripMock is a mock server for gRPC services",
  base: '/',
  lastUpdated: true,
  head: [
    [
      'script',
      { async: '', src: 'https://www.googletagmanager.com/gtag/js?id=G-2T92M9S17S' }
    ],
    [
      'script',
      {},
      `window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', 'G-2T92M9S17S');`
    ],
    [
      'link', 
      {
        rel: 'icon',
        href: 'https://github.com/bavix/gripmock/assets/5111255/b835b1a7-f572-438d-9ddb-eda7e0842db0',
        sizes: "any",
        type: "image/svg+xml",
      }
    ],
  ],
  ignoreDeadLinks: [
      /^https?:\/\/localhost:4771/,
  ],
  themeConfig: {
    // https://vitepress.dev/reference/default-theme-config
    search: {
      provider: 'algolia',
      options: {
        appId: '5F2WWL5685',
        apiKey: '4076af8af95bb73d46bb8cffa776fed1',
        indexName: 'gripmock',
      }
    },
    editLink: {
      pattern: 'https://github.com/bavix/gripmock/edit/master/docs/:path'
    },
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Guide', link: '/guide/introduction/' },
      { text: 'Issues', link: 'https://github.com/bavix/gripmock/issues' },
      { text: 'Donate', link: 'https://buymeacoffee.com/babichev' },
    ],

    sidebar: [
      {
        text: 'Getting Started',
        items: [
          { text: 'Introduction', link: '/guide/introduction' },
          { text: 'Performance Comparison', link: '/guide/introduction/performance-comparison' },
          { text: 'Quick Usage', link: '/guide/introduction/quick-usage' },
          { text: 'Environment Variables', link: '/guide/introduction/environment-variables' },
          { text: 'Advanced Usage', link: '/guide/introduction/advanced-usage' },
          { text: 'TLS and mTLS', link: '/guide/introduction/tls' },
        ],
        collapsed: false,
      },
      {
        text: 'Sources',
        items: [
          { text: 'Overview', link: '/guide/sources/' },
          { text: 'BSR', link: '/guide/sources/bsr' },
          { text: 'gRPC Reflection', link: '/guide/sources/grpc-reflection' },
        ],
        collapsed: false,
      },
      {
        text: 'Modes',
        items: [
          { text: 'Overview', link: '/guide/modes/' },
          { text: 'Proxy', link: '/guide/modes/proxy' },
          { text: 'Replay', link: '/guide/modes/replay' },
          { text: 'Capture', link: '/guide/modes/capture' },
        ],
        collapsed: false,
      },
      {
        text: 'Stubs',
        items: [
          { text: 'JSON', link: '/guide/stubs/json' },
          { text: 'YAML', link: '/guide/stubs/yaml' },
          { text: 'YAML Benefits', link: '/guide/stubs/benefits-yaml' },
          { text: 'Why IDs Matter', link: '/guide/stubs/why-ids-are-crucial' },
          { text: 'Priority', link: '/guide/stubs/priority' },
          { text: 'Times Limit', link: '/guide/stubs/times-limit' },
          { text: 'Delay', link: '/guide/stubs/delay' },
          { text: 'Output', link: '/guide/stubs/output-stream' },
          { text: 'Streaming', link: '/guide/stubs/streaming' },
          { text: 'Health Service', link: '/guide/stubs/health' },
          { text: 'Dynamic Templates', link: '/guide/stubs/dynamic-templates' },
          { text: 'Effects', link: '/guide/stubs/effects' },
          { text: 'Faker Reference', link: '/guide/stubs/faker' }
        ],
        collapsed: false,
      },
      {
        text: 'Matcher',
        items: [
          { text: 'Overview', link: '/guide/matcher/' },
          { text: 'Matching Logic', link: '/guide/matcher/logic' },
          { text: 'Input', link: '/guide/matcher/input' },
          { text: 'Headers', link: '/guide/matcher/headers' },
        ],
        collapsed: false,
      },
      {
        text: 'Schema',
        items: [
          { text: 'Overview', link: '/guide/schema/' },
          { text: 'Examples', link: '/guide/schema/examples' },
          { text: 'Validation', link: '/guide/schema/validation' },
        ],
        collapsed: false,
      },
      {
        text: 'Types',
        items: [
          { text: 'Scalar Types', link: '/guide/types/scalar-types' },
          { text: 'Well-known Types', link: '/guide/types/well-known-types' },
          { text: 'Extended Types', link: '/guide/types/extended-types' },
          { text: 'Composite & Collection Types', link: '/guide/types/composite-collection-types' },
          { text: 'Specialized Utility Types', link: '/guide/types/specialized-utility-types' },
          { text: 'Union-like Constructs', link: '/guide/types/union-like-constructs' },
        ],
        collapsed: false,
      },
      {
        text: 'Plugins',
        items: [
          { text: 'Overview', link: '/guide/plugins/' },
          { text: 'Builder Image', link: '/guide/plugins/builder-image' },
          { text: 'Advanced', link: '/guide/plugins/advanced' },
          { text: 'Testing', link: '/guide/plugins/testing' }
        ],
        collapsed: false,
      },
      {
        text: 'API',
        items: [
          { text: 'MCP API', link: '/guide/api/mcp/' },
          { text: 'Descriptors API', link: '/guide/api/descriptors' },
          {
            text: 'Stubs',
            items: [
              { text: 'Upsert', link: '/guide/api/stubs/upsert' },
              { text: 'Search', link: '/guide/api/stubs/search' },
              { text: 'List All', link: '/guide/api/stubs/list' },
              { text: 'List Used', link: '/guide/api/stubs/used-list' },
              { text: 'List Unused', link: '/guide/api/stubs/unused-list' },
              { text: 'Delete by ID', link: '/guide/api/stubs/delete' },
              { text: 'Purge All', link: '/guide/api/stubs/purge' },
            ],
            collapsed: false,
          },
          { text: 'OpenAPI', link: 'https://bavix.github.io/gripmock-openapi/' },
          { text: 'JSON Schema', link: 'https://bavix.github.io/gripmock/schema/stub.json' },
        ],
        collapsed: false,
      },
      {
        text: 'Embedded SDK',
        items: [
          { text: 'Overview', link: '/guide/embedded-sdk/' },
          { text: 'Installation', link: '/guide/embedded-sdk/installation' },
          { text: 'Quick Start', link: '/guide/embedded-sdk/quick-start' },
          { text: 'Defining Stubs', link: '/guide/embedded-sdk/defining-stubs' },
          { text: 'Health Checks', link: '/guide/embedded-sdk/health' },
          { text: 'Advanced Features', link: '/guide/embedded-sdk/advanced-features' },
          { text: 'Verification', link: '/guide/embedded-sdk/verification' },
          { text: 'Remote Mode', link: '/guide/embedded-sdk/remote-mode' },
          { text: 'Sessions', link: '/guide/embedded-sdk/sessions' },
          { text: 'Best Practices', link: '/guide/embedded-sdk/best-practices' },
        ],
        collapsed: false,
      },
      {
        text: 'CI/CD',
        items: [
          { text: 'GitHub Actions', link: '/guide/ci-cd/github-actions' },
        ],
        collapsed: false,
      },
      {
        text: 'Tooling',
        items: [
          { text: 'Check', link: '/guide/utility/check' },
          { text: 'Dump', link: '/guide/utility/dump' },
          { text: 'gRPC Testify', link: 'https://gripmock.github.io/grpctestify-rust/' },
        ],
        collapsed: false,
      },
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/bavix/gripmock' },
    ],

    footer: {
      message: 'Released under the <a href="https://github.com/bavix/gripmock/blob/master/LICENSE">MIT License</a> by <a href="https://github.com/bavix">Bavix</a> contributors.',
      copyright: 'Copyright © 2023-present <a href="https://github.com/rez1dent3">Babichev Maksim</a>',
    },
  }
})
