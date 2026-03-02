<template>
  <main>
    <h1>{{ title }}</h1>
  </main>
</template>

<script setup>
import { ref } from 'vue'

const title = ref('Pictago')
</script>

<style scoped>
main {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
}
</style>

