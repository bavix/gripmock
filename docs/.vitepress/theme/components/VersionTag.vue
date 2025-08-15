<template>
  <span 
    class="version-tag" 
    :style="versionStyle"
  >
    {{ version }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  version: {
    type: String,
    required: true
  }
})

// Simple color generation function
function generateVersionColor(version) {
  const cleanVersion = version.replace('v', '')
  const parts = cleanVersion.split('.').map(Number)
  const [major = 0, minor = 0, patch = 0] = parts
  
  // Generate colors with smooth progression from X.0.0 to X.99.99
  // Major (X) - sets the base hue for the version family
  // Minor (Y) + Patch (Z) - create smooth progression within the major version
  
  const baseHue = (major * 90) % 360 // Base hue for major version
  
  // Create smooth progression: minor * 100 + patch gives us 0-9999 range
  const progression = (minor * 100) + patch
  const maxProgression = 9999 // 99.99 * 100 + 99
  
  // Smooth hue shift within the major version family
  const hueShift = (progression / maxProgression) * 180 // Increased to 180° for maximum hue contrast
  const hue = (baseHue + hueShift) % 360
  
  // Saturation increases with progression (more "mature" versions are more saturated)
  const saturation = Math.max(20, Math.min(40 + (progression / maxProgression) * 60, 85)) // Increased minimum saturation
  
  // Lightness decreases with progression (more "mature" versions are darker)
  const lightness = Math.max(50, Math.min(70 - (progression / maxProgression) * 30, 80)) // Increased minimum lightness
  
  return { hue, saturation, lightness }
}

// Generate style with colors
const versionStyle = computed(() => {
  const color = generateVersionColor(props.version)
  
  // Check if dark theme is active (SSR safe)
  const isDark = typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
  
  if (isDark) {
    return {
      background: `hsl(${color.hue}, ${color.saturation + 15}%, ${color.lightness + 8}%)`,
      color: `hsl(${color.hue}, ${color.saturation + 15}%, 15%)`,
      border: `1px solid hsl(${color.hue}, ${color.saturation + 15}%, 75%)`
    }
  } else {
    // Light theme: much lighter backgrounds with darker text for maximum contrast
    const lightBackground = Math.max(color.lightness + 15, 75) // Much lighter backgrounds
    const lightText = Math.max(color.lightness - 45, 20) // Much darker text for readability
    
    return {
      background: `hsl(${color.hue}, ${Math.max(color.saturation - 10, 20)}%, ${lightBackground}%)`,
      color: `hsl(${color.hue}, ${Math.max(color.saturation - 10, 20)}%, ${lightText}%)`,
      border: `1px solid hsl(${color.hue}, ${Math.max(color.saturation - 10, 20)}%, 90%)`
    }
  }
})
</script>

<style scoped>
.version-tag {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  border-radius: 0.375rem;
  text-transform: uppercase;
  letter-spacing: 0.025em;
  line-height: 1;
  white-space: nowrap;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  margin: 0 0.25rem;
  vertical-align: top;
  margin-top: -0.125rem;
}

.version-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
}

/* Light theme specific styles */
:root:not(.dark) .version-tag {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(0, 0, 0, 0.1);
}

:root:not(.dark) .version-tag:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(0, 0, 0, 0.15);
}

/* Dark theme specific styles */
:root.dark .version-tag {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

:root.dark .version-tag:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
}
</style>
