import js from "@eslint/js";
import globals from "globals";
import reactHooksPlugin from "eslint-plugin-react-hooks";
import tsParser from "@typescript-eslint/parser";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import eslintConfigPrettier from "eslint-config-prettier";

export default [
  {
    ignores: ["dist/**", "node_modules/**"],
  },
  js.configs.recommended,
  {
    files: ["src/**/*.{js,jsx,ts,tsx}"],
    languageOptions: {
      parser: tsParser,
      ecmaVersion: "latest",
      sourceType: "module",
      globals: {
        ...globals.browser,
      },
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    plugins: {
      "react-hooks": reactHooksPlugin,
      "@typescript-eslint": tsPlugin,
    },
    rules: {
      ...reactHooksPlugin.configs.recommended.rules,
      // New compiler-strict lints from react-hooks v7: flag established
      // reset-state-on-prop-change / ref-in-render idioms used across the app.
      // Revisit when adopting the React Compiler; classic rules-of-hooks and
      // exhaustive-deps remain enabled.
      "react-hooks/set-state-in-effect": "off",
      "react-hooks/refs": "off",
      // TypeScript resolves identifiers/types itself; no-undef false-positives
      // on type-only usages like React.CSSProperties.
      "no-undef": "off",
      // Intentional `catch {}` — errors are surfaced through UI state instead.
      "no-empty": ["error", { allowEmptyCatch: true }],
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_" },
      ],
    },
  },
  eslintConfigPrettier,
];
