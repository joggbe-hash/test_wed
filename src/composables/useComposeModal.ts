import { ref } from 'vue'

export const showComposeModal = ref(false)

export function openComposeModal() {
  showComposeModal.value = true
}

export function closeComposeModal() {
  showComposeModal.value = false
}
