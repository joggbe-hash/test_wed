<template>
  <div class="loading-panel" role="status" aria-live="polite">
    <span class="loading-text">載入中</span>
    <div class="loading-spinner" aria-hidden="true"></div>
  </div>
</template>

<style scoped>
.loading-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  color: #4a3320;
}

.loading-text {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
}

.loading-spinner {
  width: 54px;
  height: 54px;
  border: 5px solid #e0d8cc;
  border-top-color: #4a3320;
  border-radius: 50%;
  animation: loading-spin 0.85s linear infinite;
}

@keyframes loading-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
