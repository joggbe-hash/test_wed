<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { usePageCss } from '../composables/usePageCss'

usePageCss('index.css')

const router = useRouter()
const mode = ref('login')
</script>

<template>
  <div class="auth-container">
    <div id="slider" class="form-slider" :style="{ transform: mode === 'register' ? 'translateX(-50%)' : 'translateX(0)' }">
      <div id="loginForm" class="form-section">
        <div class="input-group">
          <div class="input-label">帳號</div>
          <input type="text" class="input-field" placeholder="例如: s123456789">
        </div>

        <div class="input-group">
          <div class="input-label">密碼</div>
          <input type="password" class="input-field" placeholder="請輸入密碼">
        </div>

        <div class="forgot-password"><a href="#">忘記密碼</a></div>

        <div class="login-actions">
          <button class="btn btn-outline" @click="mode = 'register'">註冊</button>
          <button class="btn btn-primary" @click="router.push('/home')">登入</button>
        </div>
      </div>

      <div id="registerForm" class="form-section">
        <div class="reg-group"><div class="reg-label">電子信箱</div><input type="email" class="reg-input" placeholder="example@email.com"></div>
        <div class="reg-group"><div class="reg-label">使用者名稱</div><input type="text" class="reg-input" placeholder="請輸入暱稱"></div>
        <div class="reg-group"><div class="reg-label">請輸入密碼</div><input type="password" class="reg-input" placeholder="至少 8 個字元"></div>
        <div class="reg-group"><div class="reg-label">確認密碼</div><input type="password" class="reg-input" placeholder="再次輸入密碼"></div>
        <div class="reg-group"><div class="reg-label">使用者 ID</div><input type="text" class="reg-input" placeholder="例如: yourid"></div>

        <button class="reg-btn" @click="mode = 'login'">確認註冊</button>

        <div style="margin-top: 25px;">
          <a href="#" style="color: var(--nav-bg); text-decoration: none; font-weight: 500;" @click.prevent="mode = 'login'">已有帳號，回到登入</a>
        </div>
      </div>
    </div>
  </div>
</template>
