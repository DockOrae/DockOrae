<template>
  <div class="settings-page">
    <!-- 左侧分类菜单(仿 3x-ui 设置菜单) -->
    <aside class="settings-menu">
      <div class="menu-group">
        <button type="button" class="menu-item" :class="{ active: panelOpen }" @click="panelOpen = !panelOpen">
          <Icon name="settings" size="15" class="menu-icon" />
          <span class="menu-label">{{ t('settings.panelSettings') }}</span>
          <Icon name="chevronsRight" size="12" class="menu-caret" :class="{ open: panelOpen }" />
        </button>
        <Transition name="dm-expand">
          <div v-if="panelOpen" class="menu-sub">
            <button type="button" class="sub-item" :class="{ active: active === 'profile' }" @click="go('profile')">
              <span class="sub-dot" />{{ t('settings.profile') }}
            </button>
            <button type="button" class="sub-item" :class="{ active: active === 'general' }" @click="go('general')">
              <span class="sub-dot" />{{ t('settings.general') }}
            </button>
            <button type="button" class="sub-item" :class="{ active: active === 'cert' }" @click="go('cert')">
              <span class="sub-dot" />{{ t('settings.certificate') }}
            </button>
            <button type="button" class="sub-item" :class="{ active: active === 'datetime' }" @click="go('datetime')">
              <span class="sub-dot" />{{ t('settings.dateTime') }}
            </button>
          </div>
        </Transition>
      </div>
      <button type="button" class="menu-item" :class="{ active: active === 'security' }" @click="go('security')">
        <Icon name="lock" size="15" class="menu-icon" />
        <span class="menu-label">{{ t('settings.securitySettings') }}</span>
      </button>
      <button type="button" class="menu-item" :class="{ active: active === 'telegram' }" @click="go('telegram')">
        <Icon name="send" size="15" class="menu-icon" />
        <span class="menu-label">{{ t('settings.telegramBot') }}</span>
      </button>
      <button type="button" class="menu-item" :class="{ active: active === 'email' }" @click="go('email')">
        <Icon name="mail" size="15" class="menu-icon" />
        <span class="menu-label">{{ t('settings.smtpSettings') }}</span>
      </button>
      <button type="button" class="menu-item" :class="{ active: active === 'license' }" @click="go('license')">
        <Icon name="key" size="15" class="menu-icon" />
        <span class="menu-label">{{ t('license.title') }}</span>
      </button>
    </aside>

    <!-- 右侧内容区 -->
    <main class="settings-content">
      <!-- ============ 个人资料(最上面) ============ -->
      <section v-if="active === 'profile'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.profile') }}</h2>
          <div class="flex items-center gap-4 mb-5">
            <img :src="avatarPreview" alt="avatar" class="w-16 h-16 rounded-2xl object-cover ring-2 ring-white/15 shadow-lg" />
            <div class="flex-1">
              <button class="btn btn-ghost btn-sm" @click="avatarInput?.click()">
                <Icon name="image" size="13" /> {{ t('settings.' + (user.avatar ? 'changeAvatar' : 'uploadAvatar')) }}
              </button>
              <input ref="avatarInput" type="file" accept="image/jpeg,image/png,image/gif,image/webp" class="hidden" @change="uploadAvatar" />
              <p class="text-[11px] text-muted mt-1.5">{{ t('settings.avatarNote') }}</p>
            </div>
          </div>
          <form class="space-y-4" @submit.prevent="saveProfile">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label class="label">{{ t('settings.username') }}</label>
                <input v-model="profile.username" class="input" maxlength="32" />
                <p class="text-[11px] text-muted mt-1">{{ t('settings.usernameNote') }}</p>
              </div>
              <div>
                <label class="label">{{ t('settings.nickname') }}</label>
                <input v-model="profile.nickname" class="input" maxlength="32" :placeholder="t('settings.nicknamePh')" />
              </div>
            </div>
            <div v-if="profileErr" class="text-xs text-danger">{{ profileErr }}</div>
            <button type="submit" class="btn btn-brand" :disabled="profileLoading">
              <span v-if="profileLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.saveProfile') }}
            </button>
          </form>
        </div>
      </section>

      <!-- ============ 常规(仿 3x-ui GeneralTab) ============ -->
      <section v-if="active === 'general'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.general') }}</h2>

          <!-- 面板监听 IP -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webListen') }}</div>
              <div class="sr-desc">{{ t('settings.webListenDesc') }}</div>
            </div>
            <input v-model="form.webListen" class="input sr-input" :placeholder="t('settings.webListenPh')" />
          </div>

          <!-- 面板监听域名 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webDomain') }}</div>
              <div class="sr-desc">{{ t('settings.webDomainDesc') }}</div>
            </div>
            <input v-model="form.webDomain" class="input sr-input" :placeholder="t('settings.webDomainPh')" />
          </div>

          <!-- 面板监听端口 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webPort') }}</div>
              <div class="sr-desc">{{ t('settings.webPortDesc') }}</div>
            </div>
            <input v-model.number="form.webPort" type="number" class="input sr-input w-40" min="1" max="65535" />
          </div>

          <!-- URI 路径 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webBasePath') }}</div>
              <div class="sr-desc">{{ t('settings.webBasePathDesc') }}</div>
            </div>
            <input v-model="form.webBasePath" class="input sr-input" placeholder="/" />
          </div>

          <!-- 会话时长 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.sessionMaxAge') }}</div>
              <div class="sr-desc">{{ t('settings.sessionMaxAgeDesc') }}</div>
            </div>
            <div class="sr-input flex items-center gap-2 w-48">
              <input v-model.number="form.sessionMaxAge" type="number" class="input" min="1" />
              <span class="text-[12px] text-muted">{{ t('settings.minutes') }}</span>
            </div>
          </div>

          <!-- IP 限制白名单 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.ipLimitAllowlist') }}</div>
              <div class="sr-desc">{{ t('settings.ipLimitAllowlistDesc') }}</div>
            </div>
            <input v-model="allowlistText" class="input sr-input" :placeholder="t('settings.ipLimitAllowlistPh')" />
          </div>

          <div v-if="panelErr" class="text-xs text-danger mt-2">{{ panelErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="panelLoading" @click="savePanel">
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.savePanel') }}
            </button>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveNeedRestart') }}</span>
          </div>
        </div>

        <!-- 镜像加速 -->
        <div class="card p-5">
          <div class="flex items-center gap-2 mb-3">
            <Icon name="image" size="16" class="text-brand" />
            <h2 class="text-sm font-semibold">{{ t('settings.mirrors') }}</h2>
            <span v-if="mirrorPath" class="ml-auto text-[11px] text-muted truncate max-w-[220px]" :title="mirrorPath">{{ mirrorPath }}</span>
          </div>
          <p class="text-[12px] text-muted mb-3 leading-relaxed">{{ t('settings.mirrorHelper') }}</p>
          <textarea v-model="mirrorsText" class="input !h-28 code-panel font-mono text-[12px] leading-relaxed" :placeholder="t('settings.mirrorPlaceholder')" spellcheck="false" />
          <div class="flex items-center gap-3 mt-3">
            <button class="btn btn-primary btn-sm" :disabled="savingMirrors" @click="saveMirrors">
              <Icon name="save" size="13" /> {{ t('settings.saveMirrors') }}
            </button>
            <span v-if="mirrorMsg" class="text-[11px]" :class="mirrorOk ? 'text-ok' : 'text-danger'">{{ mirrorMsg }}</span>
          </div>
          <p class="text-[11px] text-muted mt-3 leading-relaxed">{{ t('settings.mirrorRestartHint') }}</p>
        </div>

        <!-- 关于 -->
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.about') }}</h2>
          <dl class="text-[13px] space-y-2">
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.panelName') }}</dt><dd class="font-medium">{{ t('app.name') }}</dd></div>
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.version') }}</dt><dd>{{ t('app.version') }}</dd></div>
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.stack') }}</dt><dd>Go (gin + Docker SDK) + Vue 3</dd></div>
            <div class="flex items-center gap-2">
              <dt class="text-muted w-28">{{ t('settings.source') }}</dt>
              <dd>
                <a href="https://github.com/MinimaxFlora/Docker_Manager_Go" target="_blank" rel="noopener" class="link flex items-center gap-1">
                  github.com/MinimaxFlora/Docker_Manager_Go <Icon name="external" size="12" />
                </a>
              </dd>
            </div>
          </dl>
        </div>
      </section>

      <!-- ============ 证书(仿 3x-ui GeneralTab 证书区) ============ -->
      <section v-if="active === 'cert'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.certificate') }}</h2>
          <p class="text-[12px] text-muted mb-4">{{ t('settings.certDesc') }}</p>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webCertFile') }}</div>
              <div class="sr-desc">{{ t('settings.webCertFileDesc') }}</div>
            </div>
            <input v-model="form.webCertFile" class="input sr-input" :placeholder="t('settings.pathPh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webKeyFile') }}</div>
              <div class="sr-desc">{{ t('settings.webKeyFileDesc') }}</div>
            </div>
            <input v-model="form.webKeyFile" class="input sr-input" :placeholder="t('settings.pathPh')" />
          </div>

          <div v-if="panelErr" class="text-xs text-danger mt-2">{{ panelErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="panelLoading" @click="savePanel">
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.savePanel') }}
            </button>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveNeedRestart') }}</span>
          </div>
        </div>
      </section>

      <!-- ============ 日期和时间(仿 3x-ui) ============ -->
      <section v-if="active === 'datetime'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.dateTime') }}</h2>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.timeZone') }}</div>
              <div class="sr-desc">{{ t('settings.timeZoneDesc') }}</div>
            </div>
            <input v-model="form.timeZone" class="input sr-input" placeholder="Asia/Shanghai" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.datePickerType') }}</div>
              <div class="sr-desc">{{ t('settings.datePickerTypeDesc') }}</div>
            </div>
            <input v-model="form.datePickerType" class="input sr-input" placeholder="gregorian" />
          </div>

          <div v-if="panelErr" class="text-xs text-danger mt-2">{{ panelErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="panelLoading" @click="savePanel">
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.savePanel') }}
            </button>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveNeedRestart') }}</span>
          </div>
        </div>
      </section>

      <!-- ============ 安全设定(仿 3x-ui SecurityTab) ============ -->
      <section v-if="active === 'security'" class="space-y-4 fade-up">
        <!-- 管理员凭证 -->
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.adminCredentials') }}</h2>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label class="label">{{ t('settings.oldUsername') }}</label>
              <input v-model="cred.oldUsername" class="input" :placeholder="t('settings.oldUsernamePh')" autocomplete="username" />
            </div>
            <div>
              <label class="label">{{ t('settings.oldPassword') }}</label>
              <input v-model="cred.oldPassword" type="password" class="input" autocomplete="current-password" />
            </div>
            <div>
              <label class="label">{{ t('settings.newUsername') }}</label>
              <input v-model="cred.newUsername" class="input" :placeholder="t('settings.newUsernamePh')" autocomplete="username" />
            </div>
            <div>
              <label class="label">{{ t('settings.newPassword') }}</label>
              <input v-model="cred.newPassword" type="password" class="input" autocomplete="new-password" />
            </div>
          </div>
          <div v-if="credErr" class="text-xs text-danger mt-2">{{ credErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="credLoading" @click="saveCredential">
              <span v-if="credLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.saveCredentials') }}
            </button>
          </div>
        </div>

        <!-- 双因素验证 -->
        <div class="card p-5">
          <div class="flex items-center gap-2 mb-4">
            <Icon name="key" size="16" class="text-brand" />
            <h2 class="text-sm font-semibold">{{ t('settings.twoFactor') }}</h2>
            <span class="ml-auto badge" :style="user.totpEnabled ? okStyle : mutedStyle">
              {{ t('settings.' + (user.totpEnabled ? 'totpEnabled' : 'totpDisabled')) }}
            </span>
          </div>
          <p class="text-[13px] text-muted mb-3">{{ t('settings.totpEnableDesc') }}</p>
          <div v-if="!user.totpEnabled">
            <div v-if="!totpSetup.uri" class="flex gap-2">
              <input v-model="totpSetup.password" type="password" class="input flex-1" :placeholder="t('settings.oldPwd')" />
              <button class="btn btn-brand" :disabled="totpBusy" @click="totpGetKey">
                <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                {{ t('settings.totpSetupBtn') }}
              </button>
            </div>
            <div v-else class="space-y-4">
              <div class="flex flex-col sm:flex-row items-center gap-5 bg-surface2/60 border border-line rounded-xl p-4">
                <img :src="totpQr" alt="QR" class="w-36 h-36 rounded-lg bg-white p-1.5" />
                <div class="flex-1 min-w-0 text-[12px] text-muted space-y-1">
                  <p>{{ t('settings.totpQrDesc') }}</p>
                  <p class="pt-1">{{ t('settings.manualKey') }}:</p>
                  <code class="block font-mono text-[11px] text-text break-all select-all">{{ totpSetup.secret }}</code>
                </div>
              </div>
              <div class="flex gap-2">
                <input v-model="totpSetup.code" class="input flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
                <button class="btn btn-brand" :disabled="totpBusy" @click="totpEnable">
                  <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  {{ t('settings.totpEnableBtn') }}
                </button>
              </div>
              <button class="text-[12px] text-muted hover:text-text" @click="totpReset">{{ t('common.cancel') }}</button>
            </div>
          </div>
          <div v-else>
            <div v-if="!disableOpen" class="flex gap-2">
              <button class="btn btn-danger" @click="disableOpen = true">{{ t('settings.totpDisableBtn') }}</button>
            </div>
            <div v-else class="space-y-3">
              <input v-model="disableForm.password" type="password" class="input" :placeholder="t('settings.oldPwd')" />
              <div class="flex gap-2">
                <input v-model="disableForm.code" class="input flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
                <button class="btn btn-danger" :disabled="totpBusy" @click="totpDisable">
                  <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  {{ t('settings.totpDisableBtn') }}
                </button>
              </div>
              <button class="text-[12px] text-muted hover:text-text" @click="disableOpen = false">{{ t('common.cancel') }}</button>
            </div>
          </div>
          <div v-if="totpErr" class="text-xs text-danger mt-2">{{ totpErr }}</div>
        </div>
      </section>

      <!-- ============ Telegram 机器人(仿 3x-ui TelegramTab) ============ -->
      <section v-if="active === 'telegram'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.telegramBot') }}</h2>
          <p class="text-[12px] text-muted mb-4">{{ t('settings.telegramDesc') }}</p>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgEnable') }}</div>
              <div class="sr-desc">{{ t('settings.tgEnableDesc') }}</div>
            </div>
            <button type="button" class="switch" :class="{ on: form.tgEnable }" @click="form.tgEnable = !form.tgEnable">
              <span class="switch-knob" />
            </button>
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgBotToken') }}</div>
              <div class="sr-desc">{{ t('settings.tgBotTokenDesc') }}</div>
            </div>
            <input v-model="form.tgBotToken" class="input sr-input" :placeholder="t('settings.tgBotTokenPh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgAdminChatId') }}</div>
              <div class="sr-desc">{{ t('settings.tgAdminChatIdDesc') }}</div>
            </div>
            <input v-model="form.tgAdminChatId" class="input sr-input" :placeholder="t('settings.tgAdminChatIdPh')" />
          </div>

          <!-- 通知事件选择 -->
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.notifyEvents') }}</div>
              <div class="sr-desc">{{ t('settings.notifyEventsDesc') }}</div>
            </div>
            <div class="event-checkboxes">
              <label v-for="ev in notifyEventOptions" :key="ev.key" class="ev-item">
                <input type="checkbox" :value="ev.key" v-model="tgEvents" />
                <span>{{ ev.label }}</span>
              </label>
            </div>
          </div>
          <div v-if="panelErr" class="text-xs text-danger mt-2">{{ panelErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="panelLoading" @click="savePanel">
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.savePanel') }}
            </button>
            <button class="btn btn-ghost" :disabled="panelLoading" @click="tgTest">{{ t('settings.tgTest') }}</button>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveOk') }}</span>
          </div>
        </div>
      </section>

      <!-- ============ 邮件(仿 3x-ui EmailTab) ============ -->
      <section v-if="active === 'email'" class="space-y-4 fade-up">
        <div class="card p-5">
          <h2 class="card-title">{{ t('settings.smtpSettings') }}</h2>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.emailEnable') }}</div>
              <div class="sr-desc">{{ t('settings.emailEnableDesc') }}</div>
            </div>
            <button type="button" class="switch" :class="{ on: form.emailEnable }" @click="form.emailEnable = !form.emailEnable">
              <span class="switch-knob" />
            </button>
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpHost') }}</div>
              <div class="sr-desc">{{ t('settings.smtpHostDesc') }}</div>
            </div>
            <input v-model="form.smtpHost" class="input sr-input" placeholder="smtp.example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpPort') }}</div>
              <div class="sr-desc">{{ t('settings.smtpPortDesc') }}</div>
            </div>
            <input v-model.number="form.smtpPort" type="number" class="input sr-input w-40" min="1" max="65535" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpUser') }}</div>
              <div class="sr-desc">{{ t('settings.smtpUserDesc') }}</div>
            </div>
            <input v-model="form.smtpUser" class="input sr-input" placeholder="user@example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpPass') }}</div>
              <div class="sr-desc">{{ t('settings.smtpPassDesc') }}</div>
            </div>
            <input v-model="form.smtpPass" type="password" class="input sr-input" :placeholder="t('settings.smtpPassPh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpFrom') }}</div>
              <div class="sr-desc">{{ t('settings.smtpFromDesc') }}</div>
            </div>
            <input v-model="form.smtpFrom" class="input sr-input" placeholder="noreply@example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.notifyEvents') }}</div>
              <div class="sr-desc">{{ t('settings.notifyEventsDesc') }}</div>
            </div>
            <div class="event-checkboxes">
              <label v-for="ev in notifyEventOptions" :key="ev.key" class="ev-item">
                <input type="checkbox" :value="ev.key" v-model="emailEvents" />
                <span>{{ ev.label }}</span>
              </label>
            </div>
          </div>

          <div v-if="panelErr" class="text-xs text-danger mt-2">{{ panelErr }}</div>
          <div class="flex items-center gap-3 mt-4">
            <button class="btn btn-brand" :disabled="panelLoading" @click="savePanel">
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              {{ t('settings.savePanel') }}
            </button>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveOk') }}</span>
          </div>
        </div>
      </section>

      <!-- ============ 许可证 ============ -->
      <section v-if="active === 'license'" class="fade-up">
        <div class="card p-5">
          <div class="flex items-center gap-2 mb-4">
            <Icon name="key" size="16" class="text-brand" />
            <h2 class="text-sm font-semibold">{{ t('license.title') }}</h2>
            <button class="btn btn-primary btn-sm ml-auto" @click="openLicForm">
              <Icon name="plus" size="13" /> {{ t('license.add') }}
            </button>
          </div>

          <div class="rounded-xl border border-line overflow-x-auto">
            <table class="table !m-0">
              <thead>
                <tr>
                  <th class="th">{{ t('license.id') }}</th>
                  <th class="th">{{ t('license.authorizedUser') }}</th>
                  <th class="th">{{ t('license.edition') }}</th>
                  <th class="th">{{ t('license.status') }}</th>
                  <th class="th">{{ t('license.boundTo') }}</th>
                  <th class="th">{{ t('license.expires') }}</th>
                  <th class="th w-36">{{ t('common.actions') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="licActive && licInfo">
                  <td class="td font-mono text-[11px]">{{ licKey.slice(0, 14) }}…</td>
                  <td class="td">{{ licInfo.user || '-' }}</td>
                  <td class="td"><span class="badge" :style="okStyle">{{ t('license.' + licInfo.type) }}</span></td>
                  <td class="td">
                    <span class="badge" :style="licInfo.status === 'expired' ? dangerStyle : okStyle">
                      {{ t('license.' + (licInfo.status === 'expired' ? 'expired' : 'active')) }}
                    </span>
                  </td>
                  <td class="td font-mono text-[11px]">{{ licDeviceId }}</td>
                  <td class="td">{{ fmtDate(licInfo.exp) }}</td>
                  <td class="td">
                    <div class="flex items-center gap-1">
                      <button class="btn btn-icon btn-sm" :title="t('license.unbind')" :disabled="licBusy" @click="deactivate">
                        <Icon name="link" size="13" />
                      </button>
                      <button class="btn btn-icon btn-sm text-danger" :title="t('license.unbindDelete')" :disabled="licBusy" @click="deactivate">
                        <Icon name="trash" size="13" />
                      </button>
                    </div>
                  </td>
                </tr>
                <tr v-else>
                  <td colspan="7" class="td text-center text-muted py-8">{{ t('license.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-if="licErr" class="text-xs text-danger mt-3">{{ licErr }}</div>

          <!-- 添加许可证弹窗 -->
          <div v-if="licFormOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4" @click.self="licFormOpen = false">
            <div class="card p-6 w-full max-w-lg shadow-2xl fade-up" style="border-width: 2px; border-color: var(--dm-line);">
              <h3 class="text-base font-semibold mb-1">{{ t('license.add') }}</h3>
              <p class="text-[12px] text-muted mb-4">{{ t('license.addHint') }}</p>
              <div
                class="rounded-xl border-2 border-dashed border-line hover:border-brand/60 transition-all duration-200 cursor-pointer flex flex-col items-center justify-center gap-2 py-12 px-8 bg-surface2/40"
                :class="{ '!border-brand/80 bg-brand/5 scale-[1.01]': licDragging }"
                @click="licFileInput?.click()"
                @dragover.prevent="licDragging = true"
                @dragleave="licDragging = false"
                @drop.prevent="onLicDrop"
              >
                <Icon name="upload" size="32" class="text-muted" />
                <p class="text-[14px] font-medium">{{ t('license.dropZone') }}</p>
                <p v-if="licFileName" class="text-[12px] text-brand font-mono">{{ licFileName }}</p>
                <p v-else class="text-[11px] text-muted">{{ t('license.dropZoneHint') }}</p>
                <input ref="licFileInput" type="file" class="hidden" accept=".lic,.key,.txt" @change="onLicFile" />
              </div>
              <div class="flex justify-end gap-2 mt-5">
                <button class="btn btn-ghost btn-sm" @click="licFormOpen = false; resetLicForm()">{{ t('common.cancel') }}</button>
                <button
                  class="btn btn-primary btn-sm transition-all duration-200"
                  :class="licFile ? 'opacity-100 shadow-lg shadow-brand/25 ring-1 ring-brand/50' : 'opacity-35 grayscale'"
                  :disabled="licBusy || !licFile"
                  @click="authorizeFile"
                >
                  <span v-if="licBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  <Icon v-else-if="licFile" name="check" size="13" />
                  {{ licFile ? t('license.authorize') : t('license.selectFileFirst') }}
                </button>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import QRCode from 'qrcode'
import Icon from '../components/Icon.vue'
import { api, setToken, getRegistryMirrors, saveRegistryMirrors, getLicense, activateLicenseFile, deactivateLicense } from '../api'
import { toastErr, toastOk } from '../toast'
import { applyUser, avatarUrl, loadLicense as refreshLicense, user } from '../store'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

// 分类切换(支持 #profile/#general/#cert/#datetime/#security/#telegram/#email/#license)
const SECTIONS = ['profile', 'general', 'cert', 'datetime', 'security', 'telegram', 'email', 'license']
const active = ref('general')
const panelOpen = ref(true)
function go(section) {
  active.value = section
  router.replace({ hash: '#' + section })
}
watch(
  () => route.hash,
  (h) => {
    const key = String(h || '').replace('#', '')
    if (SECTIONS.includes(key)) {
      active.value = key
      if (key === 'profile' || key === 'general' || key === 'cert' || key === 'datetime') panelOpen.value = true
    }
  },
  { immediate: true }
)

const okStyle = { color: '#34d399', background: 'rgba(52,211,153,.12)', border: '1px solid rgba(52,211,153,.3)' }
const mutedStyle = { color: '#8b93a7', background: 'rgba(139,147,167,.12)', border: '1px solid rgba(139,147,167,.3)' }
const dangerStyle = { color: '#f87171', background: 'rgba(248,113,113,.12)', border: '1px solid rgba(248,113,113,.3)' }
const fmtDate = (ts) => (ts ? new Date(ts * 1000).toLocaleDateString() : '-')

// 通知事件选项(对应后端 notify 类型)
const notifyEventOptions = [
  { key: 'login', label: t('notify.login') },
  { key: 'login_fail', label: t('notify.loginFail') },
  { key: 'password', label: t('notify.password') },
  { key: 'license', label: t('notify.license') },
  { key: 'container', label: t('notify.container') },
  { key: 'image', label: t('notify.image') },
  { key: 'network', label: t('notify.network') },
  { key: 'volume', label: t('notify.volume') },
]

// ---------- 面板设置 ----------
const form = reactive({
  webListen: '', webDomain: '', webPort: 8080, webBasePath: '/', sessionMaxAge: 10080,
  webCertFile: '', webKeyFile: '', timeZone: 'Asia/Shanghai', datePickerType: 'gregorian',
  tgEnable: false, tgBotToken: '', tgAdminChatId: '',
  emailEnable: false, smtpHost: '', smtpPort: 25, smtpUser: '', smtpPass: '', smtpFrom: '',
  tgNotifyEvents: [], emailNotifyEvents: [],
})
const allowlistText = ref('')
const tgEvents = ref([])
const emailEvents = ref([])
const panelLoading = ref(false)
const panelErr = ref('')
const panelSaved = ref(false)

async function loadPanelSettings() {
  try {
    const s = await api('/system/settings')
    Object.assign(form, {
      webListen: s.webListen || '',
      webDomain: s.webDomain || '',
      webPort: s.webPort || 8080,
      webBasePath: s.webBasePath || '/',
      sessionMaxAge: s.sessionMaxAge || 10080,
      webCertFile: s.webCertFile || '',
      webKeyFile: s.webKeyFile || '',
      timeZone: s.timeZone || 'Asia/Shanghai',
      datePickerType: s.datePickerType || 'gregorian',
      tgEnable: !!s.tgEnable,
      tgBotToken: s.tgBotToken || '',
      tgAdminChatId: s.tgAdminChatId || '',
      emailEnable: !!s.emailEnable,
      smtpHost: s.smtpHost || '',
      smtpPort: s.smtpPort || 25,
      smtpUser: s.smtpUser || '',
      smtpPass: s.smtpPass || '',
      smtpFrom: s.smtpFrom || '',
    })
    allowlistText.value = (s.ipLimitAllowlist || []).join(', ')
    tgEvents.value = s.tgNotifyEvents || []
    emailEvents.value = s.emailNotifyEvents || []
  } catch { /* 静默 */ }
}

async function savePanel() {
  panelErr.value = ''
  panelSaved.value = false
  const patch = {
    webListen: form.webListen.trim(),
    webDomain: form.webDomain.trim(),
    webPort: Number(form.webPort) || 8080,
    webBasePath: form.webBasePath.trim() || '/',
    sessionMaxAge: Number(form.sessionMaxAge) || 10080,
    ipLimitAllowlist: allowlistText.value.split(',').map((s) => s.trim()).filter(Boolean),
    webCertFile: form.webCertFile.trim(),
    webKeyFile: form.webKeyFile.trim(),
    timeZone: form.timeZone.trim() || 'Asia/Shanghai',
    datePickerType: form.datePickerType.trim() || 'gregorian',
    tgEnable: form.tgEnable,
    tgBotToken: form.tgBotToken.trim(),
    tgAdminChatId: form.tgAdminChatId.trim(),
    tgNotifyEvents: [...tgEvents.value],
    emailEnable: form.emailEnable,
    smtpHost: form.smtpHost.trim(),
    smtpPort: Number(form.smtpPort) || 25,
    smtpUser: form.smtpUser.trim(),
    smtpPass: form.smtpPass.trim(),
    smtpFrom: form.smtpFrom.trim(),
    emailNotifyEvents: [...emailEvents.value],
  }
  panelLoading.value = true
  try {
    const r = await api('/system/settings', { method: 'PUT', json: patch })
    panelSaved.value = true
    if (r.needRestart) toastOk(t('settings.saveNeedRestart'))
    setTimeout(() => (panelSaved.value = false), 3000)
  } catch (e) {
    panelErr.value = e.message
    toastErr(e.message)
  } finally {
    panelLoading.value = false
  }
}

async function tgTest() {
  toastOk(t('settings.tgTestSent'))
}

// ---------- 个人资料 ----------
const profile = reactive({ username: user.username || 'admin', nickname: user.nickname || '' })
watch(
  () => [user.username, user.nickname],
  ([u, n]) => {
    if (u && profile.username !== u) profile.username = u
    profile.nickname = n || ''
  }
)
const profileErr = ref('')
const profileLoading = ref(false)
const avatarInput = ref(null)
const avatarPreview = computed(() => avatarUrl() || '/logo.jpg')

async function saveProfile() {
  profileErr.value = ''
  profileLoading.value = true
  try {
    const r = await api('/profile', { method: 'POST', json: { username: profile.username.trim(), nickname: profile.nickname.trim() || null } })
    if (r.token) setToken(r.token)
    applyUser(r)
    profile.username = user.username
    profile.nickname = user.nickname || ''
    toastOk(t('settings.toastProfileSaved'))
  } catch (e) {
    profileErr.value = e.message
    toastErr(e.message)
  } finally {
    profileLoading.value = false
  }
}

async function uploadAvatar(ev) {
  const file = ev.target.files?.[0]
  ev.target.value = ''
  if (!file) return
  if (file.size > 2 * 1024 * 1024) {
    toastErr(t('settings.avatarNote'))
    return
  }
  const reader = new FileReader()
  reader.onload = async () => {
    try {
      const data = String(reader.result).split(',')[1]
      const r = await api('/avatar', { method: 'POST', json: { data } })
      user.avatar = r.avatar
      toastOk(t('settings.toastAvatarSaved'))
    } catch (e) {
      toastErr(e.message)
    }
  }
  reader.readAsDataURL(file)
}

// ---------- 管理员凭证(仿 3x-ui SecurityTab:一次提交改用户名+密码) ----------
const cred = reactive({ oldUsername: '', oldPassword: '', newUsername: '', newPassword: '' })
const credErr = ref('')
const credLoading = ref(false)

async function saveCredential() {
  credErr.value = ''
  if (!cred.oldPassword) {
    credErr.value = t('settings.pwdFillAll')
    return
  }
  // 改密码(需要旧密码校验)
  if (cred.newPassword) {
    if (cred.newPassword.length < 6) {
      credErr.value = t('settings.pwdMinLen')
      return
    }
    try {
      await api('/password', { method: 'POST', json: { old_password: cred.oldPassword, new_password: cred.newPassword } })
      user.mustChangePassword = false
    } catch (e) {
      credErr.value = e.message
      toastErr(e.message)
      return
    }
  }
  // 改用户名(需要当前用户名)
  if (cred.newUsername && cred.newUsername !== user.username) {
    try {
      const r = await api('/profile', { method: 'POST', json: { username: cred.newUsername, nickname: null } })
      if (r.token) setToken(r.token)
      applyUser(r)
    } catch (e) {
      credErr.value = e.message
      toastErr(e.message)
      return
    }
  }
  cred.oldUsername = cred.oldPassword = cred.newUsername = cred.newPassword = ''
  toastOk(t('settings.toastCredentialsSaved'))
}

// ---------- 双因素验证 ----------
const totpSetup = reactive({ password: '', uri: '', secret: '', code: '' })
const totpQr = ref('')
const totpBusy = ref(false)
const totpErr = ref('')
const disableOpen = ref(false)
const disableForm = reactive({ password: '', code: '' })

function totpReset() {
  Object.assign(totpSetup, { password: '', uri: '', secret: '', code: '' })
  totpQr.value = ''
  totpErr.value = ''
}

async function totpGetKey() {
  totpErr.value = ''
  if (!totpSetup.password) {
    totpErr.value = t('settings.pwdFillAll')
    return
  }
  totpBusy.value = true
  try {
    const r = await api('/totp/setup', { method: 'POST', json: { password: totpSetup.password } })
    totpSetup.uri = r.uri
    totpSetup.secret = r.secret
    totpQr.value = await QRCode.toDataURL(r.uri, { width: 280, margin: 1 })
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

async function totpEnable() {
  totpErr.value = ''
  if (!totpSetup.code) {
    totpErr.value = t('login.errTotpFill')
    return
  }
  totpBusy.value = true
  try {
    await api('/totp/enable', { method: 'POST', json: { code: totpSetup.code } })
    user.totpEnabled = true
    totpReset()
    toastOk(t('settings.toastTotpEnabled'))
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

async function totpDisable() {
  totpErr.value = ''
  if (!disableForm.password || !disableForm.code) {
    totpErr.value = t('settings.pwdFillAll')
    return
  }
  totpBusy.value = true
  try {
    await api('/totp/disable', { method: 'POST', json: { password: disableForm.password, code: disableForm.code } })
    user.totpEnabled = false
    disableOpen.value = false
    disableForm.password = ''
    disableForm.code = ''
    toastOk(t('settings.toastTotpDisabled'))
  } catch (e) {
    totpErr.value = e.message
  } finally {
    totpBusy.value = false
  }
}

// ---------- 镜像加速 ----------
const mirrorsText = ref('')
const mirrorPath = ref('')
const savingMirrors = ref(false)
const mirrorMsg = ref('')
const mirrorOk = ref(false)

async function loadMirrors() {
  try {
    const r = await getRegistryMirrors()
    mirrorsText.value = (r.mirrors || []).join('\n')
    mirrorPath.value = r.path || ''
  } catch { /* 静默 */ }
}

async function saveMirrors() {
  const mirrors = mirrorsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  savingMirrors.value = true
  mirrorMsg.value = ''
  try {
    await saveRegistryMirrors(mirrors)
    mirrorOk.value = true
    mirrorMsg.value = t('settings.mirrorSaved')
    setTimeout(() => (mirrorMsg.value = ''), 3000)
  } catch (e) {
    mirrorOk.value = false
    mirrorMsg.value = e.message
    toastErr(e.message)
  } finally {
    savingMirrors.value = false
  }
}

// ---------- 许可证 ----------
const licActive = ref(false)
const licInfo = ref(null)
const licKey = ref('')
const licDeviceId = ref('')
const licBusy = ref(false)
const licErr = ref('')
const licFormOpen = ref(false)
const licDragging = ref(false)
const licFileName = ref('')
const licFile = ref(null)
const licFileInput = ref(null)

async function refreshLic() {
  try {
    const r = await getLicense()
    licActive.value = !!r.active
    licInfo.value = r.info || null
    licKey.value = r.key || ''
    licDeviceId.value = r.device_id || ''
    refreshLicense()
  } catch { /* 静默 */ }
}

function openLicForm() {
  licFormOpen.value = true
  resetLicForm()
}
function resetLicForm() {
  licFileName.value = ''
  licFile.value = null
  licDragging.value = false
}
function onLicFile(ev) {
  const f = ev.target.files?.[0]
  ev.target.value = ''
  if (f) {
    licFile.value = f
    licFileName.value = f.name
  }
}
function onLicDrop(ev) {
  licDragging.value = false
  const f = ev.dataTransfer.files?.[0]
  if (f) {
    licFile.value = f
    licFileName.value = f.name
  }
}
async function authorizeFile() {
  if (!licFile.value) return
  licBusy.value = true
  licErr.value = ''
  try {
    await activateLicenseFile(licFile.value)
    licFormOpen.value = false
    resetLicForm()
    await refreshLic()
    toastOk(t('license.toastActivated'))
  } catch (e) {
    licErr.value = e.message
    toastErr(e.message)
  } finally {
    licBusy.value = false
  }
}
async function deactivate() {
  licBusy.value = true
  licErr.value = ''
  try {
    await deactivateLicense()
    await refreshLic()
    toastOk(t('license.toastDeactivated'))
  } catch (e) {
    licErr.value = e.message
  } finally {
    licBusy.value = false
  }
}

onMounted(() => {
  loadPanelSettings()
  loadMirrors()
  refreshLic()
})
</script>

<style scoped>
.settings-page {
  display: flex;
  gap: 16px;
  align-items: flex-start;
}

/* ---------- 左侧菜单(仿 3x-ui 设置菜单) ---------- */
.settings-menu {
  width: 210px;
  flex-shrink: 0;
  background: var(--dm-surface);
  border: 1px solid var(--dm-line);
  border-radius: 12px;
  padding: 8px;
  position: sticky;
  top: 0;
}
.menu-group {
  margin-bottom: 2px;
}
.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: 38px;
  padding: 0 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dm-muted);
  font-size: 13px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}
.menu-item:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}
.menu-item.active {
  color: var(--dm-brand);
  background: color-mix(in srgb, var(--dm-brand) 10%, transparent);
  font-weight: 600;
}
.menu-icon {
  flex-shrink: 0;
}
.menu-caret {
  margin-left: auto;
  transition: transform 0.2s;
  flex-shrink: 0;
}
.menu-caret.open {
  transform: rotate(90deg);
}
.menu-sub {
  padding: 2px 0 4px;
}
.sub-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 32px;
  padding: 0 12px 0 34px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--dm-muted);
  font-size: 12.5px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, color 0.15s;
}
.sub-item:hover {
  color: var(--dm-text);
  background: var(--dm-surface2);
}
.sub-item.active {
  color: var(--dm-brand);
  font-weight: 600;
}
.sub-dot {
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.5;
}

.dm-expand-enter-active,
.dm-expand-leave-active {
  transition: opacity 0.15s ease;
}
.dm-expand-enter-from,
.dm-expand-leave-to {
  opacity: 0;
}

/* ---------- 右侧内容 ---------- */
.settings-content {
  flex: 1;
  min-width: 0;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--dm-text);
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 设置行(仿 3x-ui SettingListItem) */
.setting-row {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid var(--dm-line);
}
.setting-row:last-of-type {
  border-bottom: none;
}
.sr-info {
  flex: 1;
  min-width: 0;
}
.sr-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--dm-text);
}
.sr-desc {
  font-size: 11.5px;
  color: var(--dm-muted);
  margin-top: 2px;
  line-height: 1.45;
}
.sr-input {
  width: 320px;
  max-width: 45%;
  flex-shrink: 0;
}

/* 开关(仿 3x-ui Switch) */
.switch {
  position: relative;
  width: 40px;
  height: 22px;
  border-radius: 999px;
  border: none;
  background: var(--dm-surface2);
  border: 1px solid var(--dm-line);
  cursor: pointer;
  transition: background 0.2s;
  flex-shrink: 0;
}
.switch.on {
  background: var(--dm-brand);
  border-color: var(--dm-brand);
}
.switch-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  transition: left 0.2s;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3);
}
.switch.on .switch-knob {
  left: 20px;
}

/* 通知事件多选 */
.event-checkboxes {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-width: 420px;
  justify-content: flex-end;
}
.ev-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid var(--dm-line);
  background: var(--dm-surface2);
  font-size: 12px;
  color: var(--dm-muted);
  cursor: pointer;
  user-select: none;
}
.ev-item input {
  accent-color: var(--dm-brand);
}

@media (max-width: 900px) {
  .settings-page {
    flex-direction: column;
  }
  .settings-menu {
    width: 100%;
    position: static;
    display: flex;
    flex-wrap: wrap;
    gap: 2px;
  }
  .menu-group {
    width: 100%;
  }
  .sr-input {
    width: 100%;
    max-width: none;
  }
  .setting-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
}
</style>
