<template>
  <div class="login-page">
    <!-- Left: Branding -->
    <div class="login-left">
      <div class="brand">
        <h1>Eino 智能运维</h1>
        <p>能理解、会推理、可自主行动的 AI 运维助手</p>
      </div>
    </div>

    <!-- Right: Form -->
    <div class="login-right">
      <div class="login-form-wrap">
        <h2>登录</h2>
        <el-form
          :model="loginForm"
          :rules="rules"
          ref="formRef"
          class="login-form"
          @keyup.enter="handleLogin"
        >
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="用户名"
              size="large"
            />
          </el-form-item>
          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="密码"
              size="large"
              show-password
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              size="large"
              class="login-btn"
              @click="handleLogin"
              :loading="loading"
            >
              登 录
            </el-button>
          </el-form-item>
        </el-form>
        <div class="login-hint">默认账号: admin / admin123</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from "vue";
import { useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { login } from "../api";

const router = useRouter();
const formRef = ref();
const loading = ref(false);

const loginForm = reactive({
  username: "admin",
  password: "admin123",
});

const rules = {
  username: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
};

const handleLogin = async () => {
  const valid = await formRef.value.validate().catch(() => false);
  if (!valid) return;

  loading.value = true;
  try {
    const res = await login(loginForm);
    localStorage.setItem("token", res.token);
    localStorage.setItem("username", loginForm.username);
    ElMessage.success("登录成功");
    router.push("/main");
  } catch (err) {
    ElMessage.error(err.message || "登录失败");
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-page {
  display: flex;
  min-height: 100vh;
}

.login-left {
  flex: 1;
  background: #111111;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
}

.brand {
  max-width: 420px;
}

.brand h1 {
  color: #ffffff;
  font-size: 36px;
  font-weight: 700;
  letter-spacing: 2px;
  margin: 0 0 16px 0;
}

.brand p {
  color: #999999;
  font-size: 16px;
  line-height: 1.8;
  margin: 0;
}

.login-right {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  background: #ffffff;
}

.login-form-wrap {
  width: 100%;
  max-width: 360px;
}

.login-form-wrap h2 {
  font-size: 24px;
  font-weight: 600;
  color: #111111;
  margin: 0 0 32px 0;
  letter-spacing: 1px;
}

.login-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  letter-spacing: 4px;
}

.login-hint {
  text-align: center;
  margin-top: 32px;
  color: #aaaaaa;
  font-size: 13px;
}
</style>
