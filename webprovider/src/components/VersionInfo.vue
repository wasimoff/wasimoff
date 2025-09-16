<!---------------------------------------->
<template>
  <a href="https://github.com/wasimoff/wasimoff" target="_blank">
    <button class="button">
      <span class="icon">
        <img src="../assets/github.svg" alt="GitHub Logo" />
      </span>

      <span class="info">
        github.com/wasimoff<br />
        <span class="version">
          version: <code>{{ version }}</code>
        </span>
      </span>
    </button>
  </a>
</template>

<!---------------------------------------->
<script setup lang="ts">
// @ts-ignore: defined in vite plugin
const version = VERSION;
</script>

<!---------------------------------------->
<style lang="css" scoped>
span.info {
  text-align: left;
  padding-left: 0.5em;
}

span.version {
  font-size: smaller;
}
</style>
