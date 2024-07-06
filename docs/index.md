---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
  name: GripMock
  text: Fast. Just. Comfortable.
  tagline: gRPC-MockServer
  image: https://github.com/bavix/gripmock/assets/5111255/d33740c1-2c53-4c06-a7a7-d3a9cb6e7c00
  actions:
    - theme: brand
      text: Getting started
      link: /guide/introduction/
    - theme: alt
      text: Star on GitHub ⭐
      link: https://github.com/bavix/gripmock
#
#features:
#  - title: Feature A
#    details: Lorem ipsum dolor sit amet, consectetur adipiscing elit
#  - title: Feature B
#    details: Lorem ipsum dolor sit amet, consectetur adipiscing elit
#  - title: Feature C
#    details: Lorem ipsum dolor sit amet, consectetur adipiscing elit
---

<style>
:root {
  --vp-home-hero-image-background-image: linear-gradient(-44deg, #b033ec 50%, #41b9ea 50%);
  --vp-home-hero-image-filter: blur(46px);
}

@media (min-width: 640px) {
  :root {
    --vp-home-hero-image-filter: blur(50px);
  }
}

@media (min-width: 960px) {
  :root {
    --vp-home-hero-image-filter: blur(75px);
  }
}
</style>