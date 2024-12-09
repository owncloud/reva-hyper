<script setup>
import {ref} from "vue";

const props = defineProps({
  greetings: String,
  salutation: String,
  final_word: String,
})
const greeting = ref('')
const updateGreeting = () => {
  greeting.value = `${props.greetings[Math.floor(Math.random() * props.greetings.length)]} ${props.salutation}!`
}

updateGreeting()
setInterval(updateGreeting, 1000)
</script>

<template>
  <div class="oc-flex oc-flex-middle oc-flex-center oc-height-1-1 oc-width-1-1">
    <div class="oc-text-center">
      <h1 class="oc-display-block">{{ greeting }}</h1>
      <h2 class="oc-display-block">{{ props.final_word }}</h2>
    </div>
  </div>
</template>