import DefaultTheme from 'vitepress/theme'
import './custom.css'

// Import custom components
import VersionTag from './components/VersionTag.vue'

export default {
  ...DefaultTheme,
  enhanceApp({ app }) {
    // Register custom components globally
    app.component('VersionTag', VersionTag)
  }
}
