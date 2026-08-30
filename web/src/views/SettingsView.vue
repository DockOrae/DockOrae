<template>
  <div class="settings-page">
                                     
    <main class="settings-content">
                                                                                    
      <section v-if="active === 'general' || active === 'cert' || active === 'datetime'" class="space-y-4 fade-up">
                                                         
        <Card class="p-4">
          <div class="flex items-center gap-3 flex-wrap">
            <Button variant="brand" :class="{ 'opacity-40 pointer-events-none': !settingsDirty }"
              :disabled="!settingsDirty || panelLoading"
              @click="savePanel"
            >
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              <Icon v-else name="check" size="14" /> {{ t('settings.savePanel') }}
            </button>
                                            
            <Button variant="brand"  @click="panelRestart">
              <Icon name="restart" size="14" /> {{ t('settings.restartPanel') }}
            </Button>
                                                               
            <div class="restart-hint-bar">
              <Icon name="alert" size="14" class="shrink-0" />
              <span>{{ t('settings.allChangesNeedRestart') }}</span>
            </div>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveOk') }}</span>
            <span v-if="panelErr" class="text-xs text-danger">{{ panelErr }}</span>
          </div>
        </Card>

                                           
        <div class="h-tabs">
          <button
            v-for="tab in generalTabs"
            :key="tab.key"
            type="button"
            class="h-tab"
            :class="{ active: active === tab.key }"
            @click="go(tab.key)"
          >
            <Icon :name="tab.icon" size="13" class="inline mr-1 align-[-2px]" /> {{ t(tab.labelKey) }}
          </button>
        </div>

                                      
        <Card v-if="active === 'general'" class="p-5">
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webListen') }}</div>
              <div class="sr-desc">{{ t('settings.webListenDesc') }}</div>
            </div>
            <Input v-model="form.webListen" class="sr-input" :placeholder="t('settings.webListenPh')" />
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webDomain') }}</div>
              <div class="sr-desc">{{ t('settings.webDomainDesc') }}</div>
            </div>
            <Input v-model="form.webDomain" class="sr-input" :placeholder="t('settings.webDomainPh')" />
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webPort') }}</div>
              <div class="sr-desc">{{ t('settings.webPortDesc') }}</div>
            </div>
            <Input v-model.number="form.webPort" type="number" class="sr-input w-40" min="1" max="65535" />
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webBasePath') }}</div>
              <div class="sr-desc">{{ t('settings.webBasePathDesc') }}</div>
            </div>
            <Input v-model="form.webBasePath" class="sr-input" :placeholder="t('settings.webBasePathPh')" />
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.noAuthSetting') }}</div>
              <div class="sr-desc">{{ t('settings.noAuthSettingDesc') }}</div>
            </div>
            <Select v-model="form.noAuthSetting">
              <SelectTrigger class="sr-input"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="opt in NOAUTH_OPTIONS" :key="opt.value" :value="opt.value">{{ opt.label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.sessionMaxAge') }}</div>
              <div class="sr-desc">{{ t('settings.sessionMaxAgeDesc') }}</div>
            </div>
            <div class="sr-input flex items-center gap-2 w-48">
              <Input v-model.number="form.sessionMaxAge" type="number" min="1" />
              <span class="text-[12px] text-muted">{{ t('settings.minutes') }}</span>
            </div>
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.ipLimitAllowlist') }}</div>
              <div class="sr-desc">{{ t('settings.ipLimitAllowlistDesc') }}</div>
            </div>
            <Input v-model="allowlistText" class="sr-input" :placeholder="t('settings.ipLimitAllowlistPh')" />
          </div>

                                                     
          <div class="setting-row items-start">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.mirrors') }}</div>
              <div class="sr-desc">{{ t('settings.mirrorHelper') }}</div>
              <span v-if="mirrorPath" class="block text-[11px] text-muted truncate max-w-[240px] mt-1" :title="mirrorPath">{{ mirrorPath }}</span>
            </div>
            <div class="sr-input !w-[420px]">
              <Textarea
                v-model="mirrorsText"
                rows="1"
                class="text-[12px] leading-relaxed w-full"
                style="resize: vertical; min-height: 36px; max-height: 320px"
                :placeholder="t('settings.mirrorPlaceholder')"
                spellcheck="false"
              />
            </div>
          </div>
        </Card>

                   
        <Card v-if="active === 'cert'" class="p-5">
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webCertFile') }}</div>
              <div class="sr-desc">{{ t('settings.webCertFileDesc') }}</div>
            </div>
            <Input v-model="form.webCertFile" class="sr-input" :placeholder="t('settings.pathPh')" />
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.webKeyFile') }}</div>
              <div class="sr-desc">{{ t('settings.webKeyFileDesc') }}</div>
            </div>
            <Input v-model="form.webKeyFile" class="sr-input" :placeholder="t('settings.pathPh')" />
          </div>
        </Card>

                      
        <Card v-if="active === 'datetime'" class="p-5">
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.timeZone') }}</div>
              <div class="sr-desc">{{ t('settings.timeZoneDesc') }}</div>
            </div>
            <Select v-model="form.timeZone">
              <SelectTrigger class="sr-input"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="z in TIMEZONES" :key="z" :value="z">{{ z }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.ntpServer') }}</div>
              <div class="sr-desc">{{ t('settings.ntpServerDesc') }}</div>
            </div>
            <Select v-model="form.ntpServer">
              <SelectTrigger class="sr-input"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="n in NTPSERVERS" :key="n" :value="n">{{ n }}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </Card>
      </section>

                                                                       
      <section v-if="active === 'security'" class="space-y-4 fade-up">
        <Card class="p-4">
          <div class="flex items-center gap-3 flex-wrap">
            <Button variant="brand" :class="{ 'opacity-40 pointer-events-none': !securityDirty }"
              :disabled="!securityDirty || accountLoading"
              @click="saveAccount"
            >
              <span v-if="accountLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              <Icon v-else name="check" size="14" /> {{ t('settings.saveAccount') }}
            </button>
            <Button variant="brand"  @click="panelRestart">
              <Icon name="restart" size="14" /> {{ t('settings.restartPanel') }}
            </Button>
                                                 
            <div class="restart-hint-bar">
              <Icon name="alert" size="14" class="shrink-0" />
              <span>{{ t('settings.allChangesNeedRestart') }}</span>
            </div>
            <span v-if="credErr" class="text-xs text-danger">{{ credErr }}</span>
          </div>
        </Card>

        <div class="h-tabs">
          <button
            v-for="tab in securityTabs"
            :key="tab.key"
            type="button"
            class="h-tab"
            :class="{ active: secTab === tab.key }"
            @click="secTab = tab.key"
          >
            <Icon :name="tab.icon" size="13" class="inline mr-1 align-[-2px]" /> {{ t(tab.labelKey) }}
          </button>
        </div>

                                               
        <Card v-if="secTab === 'credentials'" class="p-5">
                        
          <div class="flex items-center gap-4 mb-6">
            <img
              :src="wallpaperFailed ? '/bg.jpg' : wallpaperPreview"
              alt="wallpaper"
              class="w-36 h-24 rounded-xl object-cover ring-1 ring-line border border-line shadow"
              @error="onWallpaperError"
            />
            <div class="flex-1">
              <Button variant="ghost" size="sm"  @click="wallpaperInput?.click()">
                <Icon name="image" size="13" /> {{ t('settings.changeWallpaper') }}
              </Button>
              <input ref="wallpaperInput" type="file" accept="image/jpeg,image/png,image/gif,image/webp" class="hidden" @change="selectWallpaper" />
              <p class="text-[11px] text-muted mt-1.5">{{ t('settings.wallpaperNote') }}</p>
            </div>
          </div>

                                                         
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <Label>{{ t('settings.oldUsername') }}</label>
              <Input v-model="cred.oldUsername" autocomplete="username" />
            </div>
            <div>
              <Label>{{ t('settings.oldPassword') }}</label>
              <Input v-model="cred.oldPassword" type="password" autocomplete="current-password" />
            </div>
            <div>
              <Label>{{ t('settings.newUsername') }}</label>
              <Input v-model="cred.newUsername" :placeholder="t('settings.newUsernamePh')" autocomplete="username" />
            </div>
            <div>
              <Label>{{ t('settings.newPassword') }}</label>
              <Input v-model="cred.newPassword" type="password" :placeholder="t('settings.newPasswordPh')" autocomplete="new-password" />
            </div>
          </div>

          <div v-if="credErr" class="text-xs text-danger mt-3">{{ credErr }}</div>
          <p class="text-[11px] text-muted mt-4">{{ t('settings.saveNeedRestart') }}</p>
        </Card>

                     
        <Card v-if="secTab === 'twofactor'" class="p-5">
          <div class="flex items-center gap-2 mb-4">
            <Icon name="lock" size="16" class="text-brand" />
            <Badge :style="user.totpEnabled ? okStyle : mutedStyle">
              {{ t('settings.' + (user.totpEnabled ? 'totpEnabled' : 'totpDisabled')) }}
            </Badge>
          </div>
          <p class="text-[13px] text-muted mb-3">{{ t('settings.totpEnableDesc') }}</p>
          <div v-if="!user.totpEnabled">
            <div v-if="!totpSetup.uri" class="flex gap-2">
              <Input v-model="totpSetup.password" type="password" class="flex-1" :placeholder="t('settings.oldPwd')" />
              <Button variant="brand"  :disabled="totpBusy" @click="totpGetKey">
                <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                {{ t('settings.totpSetupBtn') }}
              </Button>
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
                <Input v-model="totpSetup.code" class="flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
                <Button variant="brand"  :disabled="totpBusy" @click="totpEnable">
                  <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  {{ t('settings.totpEnableBtn') }}
                </Button>
              </div>
              <button class="text-[12px] text-muted hover:text-text" @click="totpReset">{{ t('common.cancel') }}</button>
            </div>
          </div>
          <div v-else>
            <div v-if="!disableOpen" class="flex gap-2">
              <Button variant="destructive"  @click="disableOpen = true">{{ t('settings.totpDisableBtn') }}</Button>
            </div>
            <div v-else class="space-y-3">
              <Input v-model="disableForm.password" type="password" :placeholder="t('settings.oldPwd')" />
              <div class="flex gap-2">
                <Input v-model="disableForm.code" class="flex-1 text-center !text-base tracking-[0.4em] font-mono" maxlength="6" inputmode="numeric" :placeholder="t('settings.totpCodePh')" />
                <Button variant="destructive"  :disabled="totpBusy" @click="totpDisable">
                  <span v-if="totpBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  {{ t('settings.totpDisableBtn') }}
                </Button>
              </div>
              <button class="text-[12px] text-muted hover:text-text" @click="disableOpen = false">{{ t('common.cancel') }}</button>
            </div>
          </div>
          <div v-if="totpErr" class="text-xs text-danger mt-2">{{ totpErr }}</div>
        </Card>
      </section>

                                                                                
      <section v-if="active === 'telegram'" class="space-y-4 fade-up">
        <Card class="p-4">
          <div class="flex items-center gap-3 flex-wrap">
            <Button variant="brand" :class="{ 'opacity-40 pointer-events-none': !settingsDirty }"
              :disabled="!settingsDirty || panelLoading"
              @click="savePanel"
            >
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              <Icon v-else name="check" size="14" /> {{ t('settings.savePanel') }}
            </button>
            <Button variant="brand"  @click="panelRestart">
              <Icon name="restart" size="14" /> {{ t('settings.restartPanel') }}
            </Button>
                                                 
            <div class="restart-hint-bar">
              <Icon name="alert" size="14" class="shrink-0" />
              <span>{{ t('settings.allChangesNeedRestart') }}</span>
            </div>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveOk') }}</span>
            <span v-if="panelErr" class="text-xs text-danger">{{ panelErr }}</span>
          </div>
        </Card>

                                           
        <div class="h-tabs">
          <button type="button" class="h-tab" :class="{ active: tgTab === 'panel' }" @click="tgTab = 'panel'">
            <Icon name="settings" size="13" class="inline mr-1 align-[-2px]" /> {{ t('settings.general') }}
          </button>
          <button type="button" class="h-tab" :class="{ active: tgTab === 'notify' }" @click="tgTab = 'notify'">
            <Icon name="clock" size="13" class="inline mr-1 align-[-2px]" /> {{ t('settings.notifications') }}
          </button>
        </div>

                     
        <Card v-if="tgTab === 'panel'" class="p-5">
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
              <div class="sr-desc">{{ tgTokenConfigured && !form.tgBotToken ? t('settings.telegramTokenConfigured') : t('settings.tgBotTokenDesc', { at: '@' }) }}</div>
            </div>
            <Input v-model="form.tgBotToken" type="password" class="sr-input" :placeholder="tgTokenConfigured && !form.tgBotToken ? t('settings.telegramTokenPlaceholder') : t('settings.tgBotTokenPh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgAdminChatId') }}</div>
              <div class="sr-desc">{{ t('settings.tgAdminChatIdDesc', { at: '@' }) }}</div>
            </div>
            <Input v-model="form.tgAdminChatId" class="sr-input" :placeholder="t('settings.tgAdminChatIdPh')" />
          </div>

                                                                 
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.telegramBotLanguage') }}</div>
            </div>
            <Select v-model="form.tgLang">
              <SelectTrigger class="sr-input w-48"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem v-for="l in LANGS" :key="l.code" :value="l.code">{{ l.flag }} {{ l.label }}</SelectItem>
              </SelectContent>
            </Select>
          </div>

                                                     
          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.telegramAPIServer') }}</div>
              <div class="sr-desc">{{ t('settings.telegramAPIServerDesc') }}</div>
            </div>
            <Input v-model="form.tgBotAPIServer" class="sr-input" placeholder="https://api.example.com" />
          </div>

          <div class="mt-4">
            <Button variant="brand"  :disabled="panelLoading" @click="tgTest">{{ t('settings.tgTest') }}</Button>
          </div>
        </Card>

                                                      
        <Card v-if="tgTab === 'notify'" class="p-5">
          <div class="setting-row items-start">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.telegramNotifyTime') }}</div>
              <div class="sr-desc">{{ t('settings.telegramNotifyTimeDesc') }}</div>
            </div>
            <div class="sr-input !w-[340px]">
              <Select v-model="notifyMode" class="w-full mb-2" @update:model-value="onNotifyModeChange">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="every">{{ t('settings.notifyEvery') }}</SelectItem>
                  <SelectItem value="@hourly">{{ t('settings.notifyHourly') }}</SelectItem>
                  <SelectItem value="@daily">{{ t('settings.notifyDaily') }}</SelectItem>
                  <SelectItem value="@weekly">{{ t('settings.notifyWeekly') }}</SelectItem>
                  <SelectItem value="@monthly">{{ t('settings.notifyMonthly') }}</SelectItem>
                  <SelectItem value="custom">{{ t('settings.notifyCustom') }}</SelectItem>
                </SelectContent>
              </Select>
              <div v-if="notifyMode === 'every'" class="flex gap-2">
                <Input v-model.number="notifyNum" type="number" min="1" class="flex-1" :placeholder="t('settings.notifyInterval')" @input="onNotifyEveryChange" />
                <Select v-model="notifyUnit" class="flex-1" @update:model-value="onNotifyEveryChange">
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="minutes">{{ t('settings.unitMinutes') }}</SelectItem>
                  <SelectItem value="hours">{{ t('settings.unitHours') }}</SelectItem>
                  <SelectItem value="days">{{ t('settings.unitDays') }}</SelectItem>
                </SelectContent>
              </Select>
              </div>
              <Input v-if="notifyMode === 'custom'"
                v-model="notifyCustom"
                placeholder="0 30 8 * * *"
                @input="onNotifyCustomChange"
              />
            </div>
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgNotifyBackup') }}</div>
              <div class="sr-desc">{{ t('settings.tgNotifyBackupDesc') }}</div>
            </div>
            <button type="button" class="switch" :class="{ on: form.tgBotBackup }" @click="form.tgBotBackup = !form.tgBotBackup">
              <span class="switch-knob" />
            </button>
          </div>

          <div class="setting-row items-start">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.tgEventBusNotify') }}</div>
              <div class="sr-desc">{{ t('settings.tgEventBusNotifyDesc') }}</div>
            </div>
            <div class="notify-groups">
              <div v-for="g in notifyGroups" :key="g.key" class="ng-group">
                <div class="ng-title">
                  <Icon :name="g.icon" size="13" />
                  {{ t(g.labelKey) }}
                  <span class="ng-count">{{ countSel(g) }}/{{ g.events.length }}</span>
                </div>
                <label v-for="ev in g.events" :key="ev.key" class="ng-item">
                  <input type="checkbox" :value="ev.key" v-model="tgEvents" />
                  <span>{{ t(ev.labelKey) }}</span>
                </label>
              </div>
            </div>
          </div>
        </Card>
      </section>

                                                                         
      <section v-if="active === 'email'" class="space-y-4 fade-up">
        <Card class="p-4">
          <div class="flex items-center gap-3 flex-wrap">
            <Button variant="brand" :class="{ 'opacity-40 pointer-events-none': !settingsDirty }"
              :disabled="!settingsDirty || panelLoading"
              @click="savePanel"
            >
              <span v-if="panelLoading" class="inline-block w-4 h-4 border-2 border-white/40 border-t-white rounded-full animate-spin" />
              <Icon v-else name="check" size="14" /> {{ t('settings.savePanel') }}
            </button>
            <Button variant="brand"  @click="panelRestart">
              <Icon name="restart" size="14" /> {{ t('settings.restartPanel') }}
            </Button>
                                                 
            <div class="restart-hint-bar">
              <Icon name="alert" size="14" class="shrink-0" />
              <span>{{ t('settings.allChangesNeedRestart') }}</span>
            </div>
            <span v-if="panelSaved" class="text-[12px] text-ok">{{ t('settings.saveOk') }}</span>
            <span v-if="panelErr" class="text-xs text-danger">{{ panelErr }}</span>
          </div>
        </Card>

                                        
        <div class="h-tabs">
          <button type="button" class="h-tab" :class="{ active: emailTab === 'smtp' }" @click="emailTab = 'smtp'">
            <Icon name="mail" size="13" class="inline mr-1 align-[-2px]" /> {{ t('settings.smtpSettings') }}
          </button>
          <button type="button" class="h-tab" :class="{ active: emailTab === 'notify' }" @click="emailTab = 'notify'">
            <Icon name="clock" size="13" class="inline mr-1 align-[-2px]" /> {{ t('settings.notifications') }}
          </button>
        </div>

                        
        <Card v-if="emailTab === 'smtp'" class="p-5">
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
            <Input v-model="form.smtpHost" class="sr-input" placeholder="smtp.example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpPort') }}</div>
              <div class="sr-desc">{{ t('settings.smtpPortDesc') }}</div>
            </div>
            <Input v-model.number="form.smtpPort" type="number" class="sr-input w-40" min="1" max="65535" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpUser') }}</div>
              <div class="sr-desc">{{ t('settings.smtpUserDesc') }}</div>
            </div>
            <Input v-model="form.smtpUser" class="sr-input" placeholder="user@example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpPass') }}</div>
              <div class="sr-desc">{{ smtpPassConfigured && !form.smtpPass ? t('settings.smtpPassConfigured') : t('settings.smtpPassDesc') }}</div>
            </div>
            <Input v-model="form.smtpPass" type="password" class="sr-input" :placeholder="smtpPassConfigured && !form.smtpPass ? t('settings.smtpPassPlaceholder') : t('settings.smtpPassPh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpFrom') }}</div>
              <div class="sr-desc">{{ t('settings.smtpFromDesc') }}</div>
            </div>
            <Input v-model="form.smtpFrom" class="sr-input" placeholder="noreply@example.com" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpFromName') }}</div>
              <div class="sr-desc">{{ t('settings.smtpFromNameDesc') }}</div>
            </div>
            <Input v-model="form.smtpFromName" class="sr-input" :placeholder="t('settings.smtpFromNamePh')" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpTo') }}</div>
              <div class="sr-desc">{{ t('settings.smtpToDesc') }}</div>
            </div>
            <Input v-model="form.smtpTo" class="sr-input" :placeholder="t('settings.smtpToPh', { at: '@' })" />
          </div>

          <div class="setting-row">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpEncryption') }}</div>
              <div class="sr-desc">{{ t('settings.smtpEncryptionDesc') }}</div>
            </div>
            <Select v-model="form.smtpEncryption">
              <SelectTrigger class="sr-input w-44"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="none">{{ t('settings.encNone') }}</SelectItem>
                <SelectItem value="ssl">SSL/TLS</SelectItem>
                <SelectItem value="starttls">STARTTLS</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div class="mt-4">
            <Button variant="brand"  :disabled="panelLoading" @click="emailTest">{{ t('settings.testSmtp') }}</Button>
          </div>
        </Card>

                   
        <Card v-if="emailTab === 'notify'" class="p-5">
          <div class="setting-row items-start">
            <div class="sr-info">
              <div class="sr-label">{{ t('settings.smtpEventBusNotify') }}</div>
              <div class="sr-desc">{{ t('settings.smtpEventBusNotifyDesc') }}</div>
            </div>
            <div class="notify-groups">
              <div v-for="g in notifyGroups" :key="g.key" class="ng-group">
                <div class="ng-title">
                  <Icon :name="g.icon" size="13" />
                  {{ t(g.labelKey) }}
                  <span class="ng-count">{{ countSel(g, 'email') }}/{{ g.events.length }}</span>
                </div>
                <label v-for="ev in g.events" :key="ev.key" class="ng-item">
                  <input type="checkbox" :value="ev.key" v-model="emailEvents" />
                  <span>{{ t(ev.labelKey) }}</span>
                </label>
              </div>
            </div>
          </div>
        </Card>
      </section>

                                            
      <section v-if="active === 'license'" class="fade-up">
        <Card class="p-5">
          <div class="flex items-center gap-2 mb-4">
            <Icon name="key" size="16" class="text-brand" />
            <h2 class="text-sm font-semibold">{{ t('license.title') }}</h2>
            <Badge :style="onlineStyle" ml-auto>{{ onlineStateLabel }}</Badge>
          </div>

          <!-- 在线授权状态 -->
          <div class="rounded-xl border border-line p-4 mb-4">
            <div class="flex items-center gap-2 mb-1">
              <Icon name="link" size="14" class="text-brand" />
              <span class="text-[12px] font-medium">{{ t('license.onlineTitle') }}</span>
            </div>
            <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-muted">
              <span v-if="licOnline.sync_state" class="font-medium inline-flex items-center gap-1" :style="licSyncStyle">
                <span class="inline-block w-1.5 h-1.5 rounded-full" :style="{ background: 'currentColor' }" />
                {{ licSyncLabel }}
              </span>
              <span v-if="licOnline.last_verify">{{ t('license.lastVerify') }}: {{ fmtDateTime(licOnline.last_verify) }}</span>
              <span v-if="licOnline.grace_deadline">{{ t('license.graceDeadline') }}: {{ fmtDateTime(licOnline.grace_deadline) }}</span>
              <span v-if="licOnline.last_event_id" class="font-mono opacity-70">{{ t('license.lastEvent') }}: {{ licOnline.last_event_id }}</span>
              <Button variant="ghost" size="sm" class="ml-auto" v-if="licKey"
                :disabled="licBusy"
                @click="verifyNow"
              >
                <span v-if="licBusy" class="inline-block w-3 h-3 border-2 border-white/40 border-t-white rounded-full animate-spin mr-1" />
                {{ licBusy ? t('license.verifying') : t('license.verifyNow') }}
              </Button>
            </div>
          </div>

          <!-- ============ 当前许可证(一个实例最多一个 License) ============ -->
          <!-- 已激活 -->
          <div v-if="licActive && licInfo" class="rounded-xl border border-ok/25 bg-ok/5 p-5">
            <div class="flex items-center gap-2.5 mb-4">
              <span class="w-9 h-9 rounded-full bg-ok/15 text-ok flex items-center justify-center">
                <Icon name="check" size="16" />
              </span>
              <div>
                <div class="text-[14px] font-semibold text-text leading-tight">{{ t('license.active') }}</div>
                <div class="text-[11px] text-ok/80">{{ t('license.onlineVerified') }}</div>
              </div>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-3">
              <div>
                <div class="text-[11px] text-muted mb-0.5">{{ t('license.licenseId') }}</div>
                <div class="font-mono text-[13px] text-text">{{ licInfo.license_id || licKey.slice(0, 20) }}…</div>
              </div>
              <div>
                <div class="text-[11px] text-muted mb-0.5">{{ t('license.edition') }}</div>
                <div class="text-[13px] text-text">{{ t('license.' + (licInfo.plan || licInfo.type || 'pro')) }}</div>
              </div>
              <div>
                <div class="text-[11px] text-muted mb-0.5">{{ t('license.authorizedUser') }}</div>
                <div class="text-[13px] text-text">{{ licInfo.customer || licInfo.user || '-' }}</div>
              </div>
              <div>
                <div class="text-[11px] text-muted mb-0.5">{{ t('license.expires') }}</div>
                <div class="text-[13px] text-text">{{ fmtDate(licInfo.expires_at || licInfo.exp) }}</div>
              </div>
              <div class="sm:col-span-2">
                <div class="text-[11px] text-muted mb-0.5">{{ t('license.deviceId') }}</div>
                <div class="font-mono text-[12px] text-text">{{ licDeviceId }}</div>
              </div>
            </div>
            <div class="flex items-center gap-2 mt-5 pt-4 border-t border-ok/15">
              <Button variant="ghost" size="sm" class="!text-warn" :disabled="licBusy" @click="confirmUnbind">
                <Icon name="link" size="13" /> {{ t('license.unbind') }}
              </Button>
              <span class="text-[11px] text-muted">{{ t('license.unbindHint') }}</span>
            </div>
          </div>

          <!-- 已吊销(仅吊销场景显示"已吊销") -->
          <div v-else-if="licRevoked" class="rounded-xl border border-danger/30 bg-danger/10 p-5">
            <div class="flex items-center gap-2.5 mb-3">
              <span class="w-9 h-9 rounded-full bg-danger/15 text-danger flex items-center justify-center">
                <Icon name="x" size="16" />
              </span>
              <div>
                <div class="text-[14px] font-semibold text-danger leading-tight">{{ t('license.revoked') }}</div>
                <div class="text-[11px] text-muted">{{ t('license.revokedDesc') }}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <Button variant="brand" size="sm" @click="openLicForm">
                <Icon name="plus" size="13" /> {{ t('license.reactivate') }}
              </Button>
            </div>
          </div>

          <!-- 管理员强制解绑(明确提示:请重新激活,不是吊销) -->
          <div v-else-if="licAdminUnbound" class="rounded-xl border border-warn/30 bg-warn/10 p-5">
            <div class="flex items-center gap-2.5 mb-3">
              <span class="w-9 h-9 rounded-full bg-warn/15 text-warn flex items-center justify-center">
                <Icon name="link" size="16" />
              </span>
              <div>
                <div class="text-[14px] font-semibold text-warn leading-tight">{{ t('license.adminUnboundTitle') }}</div>
                <div class="text-[12px] text-text mt-0.5">{{ t('license.adminUnboundDesc') }}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <Button variant="brand" size="sm" @click="openLicForm">
                <Icon name="plus" size="13" /> {{ t('license.reactivate') }}
              </Button>
              <span v-if="licKey" class="text-[11px] text-muted">{{ t('license.keyKept') }}</span>
            </div>
          </div>

          <!-- 未激活 / 已解绑(普通) -->
          <div v-else class="rounded-xl border border-line bg-surface2/40 p-5">
            <div class="flex items-center gap-2.5 mb-3">
              <span class="w-9 h-9 rounded-full bg-surface2 text-muted flex items-center justify-center">
                <Icon name="key" size="15" />
              </span>
              <div>
                <div class="text-[14px] font-semibold text-muted leading-tight">{{ t('license.inactive') }}</div>
                <div class="text-[12px] text-muted mt-0.5">{{ t('license.unboundDesc') }}</div>
              </div>
            </div>
            <div class="flex items-center gap-2">
              <Button variant="brand" size="sm" @click="openLicForm">
                <Icon name="plus" size="13" /> {{ t('license.activate') }}
              </Button>
              <span v-if="licKey" class="text-[11px] text-muted">{{ t('license.keyKept') }}</span>
            </div>
          </div>

          <div v-if="licErr" class="text-xs text-danger mt-3">{{ licErr }}</div>

          <div v-if="licFormOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4" @click.self="licFormOpen = false">
            <Card class="p-6 w-full max-w-lg shadow-2xl fade-up" style="border-width: 2px; border-color: var(--dm-line);">
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
                <Button variant="ghost" size="sm"  @click="licFormOpen = false; resetLicForm()">{{ t('common.cancel') }}</Button>
                <Button variant="brand" size="sm" class="transition-all duration-200" :class="licFile ? 'opacity-100 shadow-lg shadow-brand/25 ring-1 ring-brand/50' : 'opacity-35 grayscale'"
                  :disabled="licBusy || !licFile"
                  @click="authorizeFile"
                >
                  <span v-if="licBusy" class="inline-block w-3.5 h-3.5 border-2 border-white/40 border-t-white rounded-full animate-spin" />
                  <Icon v-else-if="licFile" name="check" size="13" />
                  {{ licFile ? t('license.authorize') : t('license.selectFileFirst') }}
                </button>
              </div>
            </Card>
          </div>
        </Card>
      </section>

                                                                 
      <section v-if="active === 'about'" class="fade-up">
        <Card class="p-5">
          <h2 class="card-title">{{ t('settings.about') }}</h2>
          <dl class="text-[13px] space-y-2">
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.panelName') }}</dt><dd class="font-medium">{{ t('app.name') }}</dd></div>
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.version') }}</dt><dd>{{ t('app.version') }}</dd></div>
            <div class="flex"><dt class="text-muted w-28">{{ t('settings.stack') }}</dt><dd>Go (gin + Docker SDK) + Vue 3</dd></div>
            <div class="flex items-center gap-2">
              <dt class="text-muted w-28">{{ t('settings.source') }}</dt>
              <dd>
                <a href="https://github.com/DockOrae/DockOrae" target="_blank" rel="noopener" class="link flex items-center gap-1">
                  github.com/DockOrae/DockOrae <Icon name="external" size="12" />
                </a>
              </dd>
            </div>
          </dl>
        </Card>
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
import { LANGS } from '../i18n'
import { api, setToken, getRegistryMirrors, saveRegistryMirrors, getLicense, activateLicenseFile, deactivateLicense, verifyLicense } from '../api'
import { toastErr, toastOk } from '../toast'
import { useConfirm } from '../confirm'
import { applyUser, loadLicense as refreshLicense, licenseActive as licActive, licenseInfo as licInfo, licenseOnline, user } from '../store'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const { t } = useI18n()
const confirmAction = useConfirm()
const route = useRoute()
const router = useRouter()

// 分类切换(#general/#security/#telegram/#email/#license/#about;常规页内 tab:#general/#cert/#datetime)
const SECTIONS = ['general', 'cert', 'datetime', 'security', 'telegram', 'email', 'license', 'about']
// 常规页内横向 tab(仿 3x-ui GeneralTab)
const generalTabs = [
  { key: 'general', labelKey: 'settings.general', icon: 'settings' },
  { key: 'cert', labelKey: 'settings.certificate', icon: 'shield' },
  { key: 'datetime', labelKey: 'settings.dateTime', icon: 'clock' },
]
// 安全设定页内横向 tab(仿 3x-ui SecurityTab)
const securityTabs = [
  { key: 'credentials', labelKey: 'settings.adminCredentials', icon: 'user' },
  { key: 'twofactor', labelKey: 'settings.twoFactor', icon: 'lock' },
]
const active = ref('general')
const secTab = ref('credentials')
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
    }
  },
  { immediate: true }
)

// 时区选择列表(常用时区)
const TIMEZONES = [
  'UTC',
  'Asia/Shanghai', 'Asia/Hong_Kong', 'Asia/Taipei', 'Asia/Singapore', 'Asia/Tokyo',
  'Asia/Seoul', 'Asia/Kolkata', 'Asia/Dubai', 'Asia/Bangkok', 'Asia/Jakarta', 'Asia/Manila',
  'Europe/London', 'Europe/Paris', 'Europe/Berlin', 'Europe/Moscow',
  'America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles',
  'America/Sao_Paulo', 'Australia/Sydney', 'Australia/Perth',
]

// NTP 时间同步服务器选择列表
const NTPSERVERS = [
  'pool.ntp.org', 'time.apple.com', 'time1.google.com', 'time.cloudflare.com',
  'ntp.tencent.com', 'ntp1.aliyun.com', 'ntp.ntsc.ac.cn', 'cn.ntp.org.cn',
]

// 重启面板(设置保存后需重启生效)
function panelRestart() {
  confirmAction(t('status.restartConfirm'), { title: t('status.restart'), danger: true, confirmText: t('common.restart') })
    .then((ok) => {
      if (!ok) return
      api('/system/restart', { method: 'POST' })
        .then(() => toastOk(t('status.restarting')))
        .catch((e) => toastErr(e.message))
    })
}

const okStyle = { color: '#34d399', background: 'rgba(52,211,153,.12)', border: '1px solid rgba(52,211,153,.3)' }
const mutedStyle = { color: '#8b93a7', background: 'rgba(139,147,167,.12)', border: '1px solid rgba(139,147,167,.3)' }
const dangerStyle = { color: '#f87171', background: 'rgba(248,113,113,.12)', border: '1px solid rgba(248,113,113,.3)' }
const fmtDate = (ts) => (ts ? new Date(ts * 1000).toLocaleDateString() : '-')
// V2 Key 的功能列表展示(与 License Server Feature Registry 一致)
const fmtFeatures = (feats) => {
  if (!Array.isArray(feats) || feats.length === 0) return '-'
  return feats.join(', ')
}

// 子 tab 状态(仿 3x-ui TelegramTab/EmailTab:面板设置|通知、SMTP 设置|通知)
const tgTab = ref('panel')
const emailTab = ref('smtp')

// 通知事件分组(仿 3x-ui NotificationGroup:分组标题 + 计数 + 事件勾选)
const notifyGroups = [
  {
    key: 'account', icon: 'user', labelKey: 'notify.groupAccount',
    events: [
      { key: 'login', labelKey: 'notify.login' },
      { key: 'login_fail', labelKey: 'notify.loginFail' },
      { key: 'password', labelKey: 'notify.password' },
      { key: 'license', labelKey: 'notify.license' },
      { key: 'system', labelKey: 'notify.system' },
    ],
  },
  {
    key: 'docker', icon: 'container', labelKey: 'notify.groupDocker',
    events: [
      { key: 'container', labelKey: 'notify.container' },
      { key: 'image', labelKey: 'notify.image' },
      { key: 'network', labelKey: 'notify.network' },
      { key: 'volume', labelKey: 'notify.volume' },
    ],
  },
]
// 某分组已勾选数量(仿 3x-ui NotificationHeader count/total)
function countSel(g, kind = 'tg') {
  const list = kind === 'email' ? emailEvents.value : tgEvents.value
  return g.events.filter((e) => list.includes(e.key)).length
}

// ---------- Telegram 周期报告时间(仿 3x-ui NotifyTimeField) ----------
const notifyMode = ref('every') // every | @hourly | @daily | @weekly | @monthly | custom
const notifyNum = ref(1)
const notifyUnit = ref('m')
const notifyCustom = ref('')
const EVERY_RE = /^@every\s+(\d+)\s*([smh])$/i

function parseNotifyTime(raw) {
  const v = (raw || '').trim()
  const m = v.match(EVERY_RE)
  if (m) {
    notifyMode.value = 'every'
    notifyNum.value = Math.max(1, Number(m[1]) || 1)
    notifyUnit.value = m[2].toLowerCase()
    notifyCustom.value = ''
    return
  }
  if (['@hourly', '@daily', '@weekly', '@monthly'].includes(v)) {
    notifyMode.value = v
    notifyNum.value = 1
    notifyUnit.value = 'h'
    notifyCustom.value = ''
    return
  }
  notifyMode.value = 'custom'
  notifyNum.value = 1
  notifyUnit.value = 'h'
  notifyCustom.value = v
}

function composeNotifyTime() {
  if (notifyMode.value === 'every') return `@every ${Math.max(1, notifyNum.value || 1)}${notifyUnit.value}`
  if (notifyMode.value === 'custom') return notifyCustom.value
  return notifyMode.value
}

function onNotifyEveryChange() {
  form.tgRunTime = composeNotifyTime()
}
function onNotifyModeChange() {
  // 切到自定义时,用当前选择的 crontab 等价表达式做种子(仿 3x-ui toCrontab)
  if (notifyMode.value === 'custom' && !notifyCustom.value.trim()) {
    notifyCustom.value = toCrontab()
  }
  form.tgRunTime = composeNotifyTime()
}
function onNotifyCustomChange() {
  form.tgRunTime = notifyCustom.value
}
function toCrontab() {
  switch (notifyMode.value) {
    case '@hourly': return '0 0 * * * *'
    case '@daily': return '0 0 0 * * *'
    case '@weekly': return '0 0 0 * * 0'
    case '@monthly': return '0 0 0 1 * *'
    case 'every': {
      const n = Math.max(1, notifyNum.value || 1)
      if (notifyUnit.value === 's') return `*/${n} * * * * *`
      if (notifyUnit.value === 'm') return `0 */${n} * * * *`
      return `0 0 */${n} * * *`
    }
    default: return notifyCustom.value
  }
}

// ---------- 面板设置 ----------
const form = reactive({
  webListen: '', webDomain: '', webPort: 8080, webBasePath: '/', noAuthSetting: '401', sessionMaxAge: 10080,
  webCertFile: '', webKeyFile: '', webForceSSL: false, timeZone: 'Asia/Shanghai', datePickerType: 'gregorian', ntpServer: 'pool.ntp.org',
  tgEnable: false, tgBotToken: '', tgAdminChatId: '', tgRunTime: '', tgBotBackup: false,
  tgLang: '', tgBotAPIServer: '',
  emailEnable: false, smtpHost: '', smtpPort: 25, smtpUser: '', smtpPass: '', smtpFrom: '',
  smtpFromName: '', smtpTo: '', smtpEncryption: 'none',
  tgNotifyEvents: [], emailNotifyEvents: [],
})
const allowlistText = ref('')
const tgEvents = ref([])
const emailEvents = ref([])
const panelLoading = ref(false)
const panelErr = ref('')
const panelSaved = ref(false)

// 未认证设置选项(仿 1Panel noAuthSetting)
const NOAUTH_OPTIONS = ['200', '400', '401', '403', '404', '408', '416', '444', '500'].map((v) => ({
  value: v,
  label: `${v} - ${t('settings.noAuth' + v)}`,
}))
// 服务器端已配置的密钥(脱敏后非空):输入框留空,避免把脱敏值回传覆盖真值
const tgTokenConfigured = ref(false)
const smtpPassConfigured = ref(false)

async function loadPanelSettings() {
  try {
    const s = await api('/system/settings')
    Object.assign(form, {
      webListen: s.webListen || '',
      webDomain: s.webDomain || '',
      webPort: s.webPort || 8080,
      webBasePath: s.webBasePath || '/',
      noAuthSetting: s.noAuthSetting || '401',
      sessionMaxAge: s.sessionMaxAge || 10080,
      webCertFile: s.webCertFile || '',
      webKeyFile: s.webKeyFile || '',
      webForceSSL: !!s.webForceSSL,
      timeZone: s.timeZone || 'Asia/Shanghai',
      datePickerType: s.datePickerType || 'gregorian',
      ntpServer: s.ntpServer || 'pool.ntp.org',
      tgEnable: !!s.tgEnable,
      tgBotToken: '', // 已配置时留空:避免把脱敏值回传覆盖真 token
      tgAdminChatId: s.tgAdminChatId || '',
      tgRunTime: s.tgRunTime || '',
      tgBotBackup: !!s.tgBotBackup,
      tgLang: s.tgLang || '',
      tgBotAPIServer: s.tgBotAPIServer || '',
      emailEnable: !!s.emailEnable,
      smtpHost: s.smtpHost || '',
      smtpPort: s.smtpPort || 25,
      smtpUser: s.smtpUser || '',
      smtpPass: '', // 已配置时留空:避免把脱敏值回传覆盖真密码
      smtpFrom: s.smtpFrom || '',
      smtpFromName: s.smtpFromName || '',
      smtpTo: s.smtpTo || '',
      smtpEncryption: s.smtpEncryption || 'none',
    })
    allowlistText.value = (s.ipLimitAllowlist || []).join(', ')
    tgEvents.value = s.tgNotifyEvents || []
    emailEvents.value = s.emailNotifyEvents || []
    tgTokenConfigured.value = !!s.tgBotToken
    smtpPassConfigured.value = !!s.smtpPass
    parseNotifyTime(s.tgRunTime || '')
    baseSnapshot = snapshotForm()
    settingsDirty.value = false
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
    noAuthSetting: form.noAuthSetting || '401',
    sessionMaxAge: Number(form.sessionMaxAge) || 10080,
    ipLimitAllowlist: allowlistText.value.split(',').map((s) => s.trim()).filter(Boolean),
    webCertFile: form.webCertFile.trim(),
    webKeyFile: form.webKeyFile.trim(),
    webForceSSL: form.webForceSSL,
    timeZone: form.timeZone.trim() || 'Asia/Shanghai',
    datePickerType: form.datePickerType.trim() || 'gregorian',
    ntpServer: form.ntpServer.trim() || 'pool.ntp.org',
    tgEnable: form.tgEnable,
    tgBotToken: form.tgBotToken.trim(),
    tgAdminChatId: form.tgAdminChatId.trim(),
    tgRunTime: form.tgRunTime.trim(),
    tgBotBackup: form.tgBotBackup,
    tgLang: form.tgLang,
    tgBotAPIServer: form.tgBotAPIServer.trim(),
    tgNotifyEvents: [...tgEvents.value],
    emailEnable: form.emailEnable,
    smtpHost: form.smtpHost.trim(),
    smtpPort: Number(form.smtpPort) || 25,
    smtpUser: form.smtpUser.trim(),
    smtpPass: form.smtpPass.trim(),
    smtpFrom: form.smtpFrom.trim(),
    smtpFromName: form.smtpFromName.trim(),
    smtpTo: form.smtpTo.trim(),
    smtpEncryption: form.smtpEncryption || 'none',
    emailNotifyEvents: [...emailEvents.value],
  }
  panelLoading.value = true
  try {
    await api('/system/settings', { method: 'PUT', json: patch })
    // 镜像加速并入顶部保存(不再独立保存按钮)
    try {
      await saveRegistryMirrors(mirrorsText.value.split('\n').map((s) => s.trim()).filter(Boolean))
    } catch { /* 镜像保存失败不阻断面板设置保存 */ }
    panelSaved.value = true
    baseSnapshot = snapshotForm()
    settingsDirty.value = false
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

// 测试邮件(仿 3x-ui testSmtp:后端实际发一封测试邮件)
async function emailTest() {
  try {
    await api('/system/test-email', { method: 'POST' })
    toastOk(t('settings.testSmtpSent'))
  } catch (e) {
    toastErr(e.message)
  }
}

// ---------- 账号凭证(原用户名/原密码/新用户名/新密码;原用户名预填当前账号) ----------
const cred = reactive({ oldUsername: user.username || '', oldPassword: '', newUsername: '', newPassword: '' })
const credErr = ref('')
const accountLoading = ref(false)

async function saveAccount() {
  credErr.value = ''
  const u = cred.oldUsername.trim()
  const nu = cred.newUsername.trim()
  const np = cred.newPassword
  const op = cred.oldPassword
  if (!nu && !np) {
    credErr.value = t('settings.credNothingToChange')
    return
  }
  // 原用户名必须等于当前登录用户名(它是身份验证字段,不是要改的名字)
  if (u && u !== user.username) {
    credErr.value = t('settings.errOldUsername')
    return
  }
  if (np && !op) {
    credErr.value = t('settings.pwdFillAll')
    return
  }
  if (np && np.length < 6) {
    credErr.value = t('settings.pwdMinLen')
    return
  }
  accountLoading.value = true
  try {
    if (np) {
      await api('/password', { method: 'POST', json: { old_password: op, new_password: np } })
      user.mustChangePassword = false
    }
    if (nu && nu !== user.username) {
      const r = await api('/profile', { method: 'POST', json: { username: nu, nickname: null } })
      if (r.token) setToken(r.token)
      applyUser(r)
      cred.oldUsername = nu
      cred.newUsername = ''
    }
    if (wallpaperPending.value) {
      await saveWallpaper()
      toastOk(t('settings.toastWallpaperSaved'))
    }
    cred.oldPassword = cred.newUsername = cred.newPassword = ''
    credBase = credSnapshot()
    toastOk(t('settings.toastCredentialsSaved'))
  } catch (e) {
    credErr.value = e.message
    toastErr(e.message)
  } finally {
    accountLoading.value = false
  }
}

// ---------- 登录页壁纸(替换登录背景图) ----------
const wallpaperInput = ref(null)
const wallpaperPreview = ref('/api/system/wallpaper?t=' + Date.now())
const wallpaperFailed = ref(false)
// 新选的壁纸(base64),随"保存"按钮一起上传
const wallpaperPending = ref('')

// 未上传自定义壁纸时回退内置 /bg.jpg(同登录页)
function onWallpaperError() {
  wallpaperFailed.value = true
}
function refreshWallpaperPreview() {
  wallpaperFailed.value = false
  wallpaperPreview.value = '/api/system/wallpaper?t=' + Date.now()
}
// 选图仅做本地预览并标记 dirty,真正上传在 saveAccount 里
function selectWallpaper(ev) {
  const file = ev.target.files?.[0]
  ev.target.value = ''
  if (!file) return
  if (file.size > 10 * 1024 * 1024) {
    toastErr(t('settings.wallpaperNote'))
    return
  }
  const reader = new FileReader()
  reader.onload = () => {
    const data = String(reader.result).split(',')[1]
    wallpaperPending.value = data
    wallpaperPreview.value = String(reader.result)
  }
  reader.readAsDataURL(file)
}
async function saveWallpaper() {
  if (!wallpaperPending.value) return
  await api('/system/wallpaper', { method: 'POST', json: { data: wallpaperPending.value } })
  wallpaperPending.value = ''
  refreshWallpaperPreview()
}

// 安全页保存按钮 dirty 检测(凭证字段有输入或新选了壁纸才亮)
const securityDirty = ref(false)
let credBase = ''
function credSnapshot() {
  return JSON.stringify({ u: cred.oldUsername, p: cred.oldPassword, nu: cred.newUsername, np: cred.newPassword })
}
watch([cred, wallpaperPending], () => {
  securityDirty.value = credSnapshot() !== credBase || !!wallpaperPending.value
}, { deep: true })

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

// ---------- 镜像加速(保存并入顶部"保存"按钮) ----------
const mirrorsText = ref('')
const mirrorPath = ref('')

// 保存按钮 dirty 检测(仿 3x-ui:有修改保存才亮,保存后长条提示重启)
const settingsDirty = ref(false)
let baseSnapshot = ''
function snapshotForm() {
  return JSON.stringify({
    ...form,
    allowlist: allowlistText.value,
    mirrors: mirrorsText.value,
    tgEvents: [...tgEvents.value].sort(),
    emailEvents: [...emailEvents.value].sort(),
  })
}
watch([form, allowlistText, mirrorsText, tgEvents, emailEvents], () => {
  settingsDirty.value = snapshotForm() !== baseSnapshot
  if (settingsDirty.value) panelSaved.value = false // 又有修改:长条提示消失,等待再次保存
}, { deep: true })

async function loadMirrors() {
  try {
    const r = await getRegistryMirrors()
    mirrorsText.value = (r.mirrors || []).join('\n')
    mirrorPath.value = r.path || ''
  } catch { /* 静默 */ }
}

// ---------- 许可证 ----------
// 激活状态/详情绑定 store 的 ref(/ws/license 推送后表格实时更新,无需刷新页面)
const licKey = ref('')
const licDeviceId = ref('')
const licBusy = ref(false)
const licErr = ref('')
const licFormOpen = ref(false)
const licDragging = ref(false)
const licFileName = ref('')
const licFile = ref(null)
const licFileInput = ref(null)
// 在线状态(V3):直接绑定 store 的 ref,/ws/license 实时推送自动更新(无需刷新页面)
const licOnline = licenseOnline // { mode, state, sync_state, last_verify, grace_deadline, verify_state, last_event_id, state_version }

// V3 同步状态(sync_state)展示:online/offline/grace/grace_expired/server_recovered/revoked/blocked
const licSyncStyle = computed(() => {
  switch (licOnline.value.sync_state) {
    case 'online': return okStyle
    case 'server_recovered': return { color: '#38bdf8', background: 'rgba(56,189,248,.12)', border: '1px solid rgba(56,189,248,.3)' }
    case 'grace':
    case 'offline': return { color: '#fbbf24', background: 'rgba(251,191,36,.12)', border: '1px solid rgba(251,191,36,.3)' }
    case 'grace_expired':
    case 'revoked':
    case 'blocked': return dangerStyle
    default: return mutedStyle
  }
})
const licSyncLabel = computed(() => {
  switch (licOnline.value.sync_state) {
    case 'online': return t('license.syncOnline')
    case 'server_recovered': return t('license.syncRecovered')
    case 'grace': return t('license.syncGrace')
    case 'offline': return t('license.syncOffline')
    case 'grace_expired': return t('license.syncGraceExpired')
    case 'revoked': return t('license.syncRevoked')
    case 'blocked': return t('license.syncBlocked')
    default: return ''
  }
})

// 表格许可证状态列:吊销/封禁/过期优先于已激活(WS 实时更新)
const licStatusLabel = computed(() => {
  const vs = licOnline.value.verify_state
  if (vs === 'revoked' || vs === 'invalid' || licOnline.value.sync_state === 'revoked') return t('license.revoked')
  if (vs === 'blocked' || licOnline.value.sync_state === 'blocked') return t('license.blocked')
  if (licInfo.value && licInfo.value.status === 'expired') return t('license.expired')
  if (vs === 'clock_rollback') return t('license.clockRollback')
  return t('license.active')
})
const licStatusStyle = computed(() => {
  const vs = licOnline.value.verify_state
  if (vs === 'revoked' || vs === 'invalid' || vs === 'blocked' || vs === 'clock_rollback'
    || licOnline.value.sync_state === 'revoked' || licOnline.value.sync_state === 'blocked') return dangerStyle
  if (licInfo.value && licInfo.value.status === 'expired') return dangerStyle
  return okStyle
})

// 在线验证状态展示(style + label)
const onlineStyle = computed(() => {
  switch (licOnline.value.state) {
    case 'verified': return okStyle
    case 'grace': return { color: '#fbbf24', background: 'rgba(251,191,36,.12)', border: '1px solid rgba(251,191,36,.3)' }
    case 'grace_expired': return dangerStyle
    case 'revoked': return dangerStyle
    case 'version_blocked': return dangerStyle
    case 'clock_rollback': return dangerStyle
    case 'update_required': return { color: '#fbbf24', background: 'rgba(251,191,36,.12)', border: '1px solid rgba(251,191,36,.3)' }
    default: return mutedStyle
  }
})
const onlineStateLabel = computed(() => {
  switch (licOnline.value.state) {
    case 'verified': return t('license.onlineVerified')
    case 'grace': return t('license.onlineGrace')
    case 'grace_expired': return t('license.onlineGraceExpired')
    case 'revoked': return t('license.onlineRevoked')
    case 'version_blocked': return t('license.onlineVersionBlocked')
    case 'clock_rollback': return t('license.onlineClockRollback')
    case 'update_required': return t('license.onlineUpdateRequired')
    case 'never': return t('license.onlineNever')
    default: return t('license.offlineMode')
  }
})

function fmtDateTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

async function refreshLic() {
  try {
    const r = await getLicense()
    licActive.value = !!r.active
    licInfo.value = r.info || null
    licKey.value = r.key || ''
    licDeviceId.value = r.device_id || ''
    licOnline.value = r.online || {}
    refreshLicense()
  } catch { /* 静默 */ }
}

// 手动触发一次在线验证(吊销即时触达)
async function verifyNow() {
  licBusy.value = true
  licErr.value = ''
  try {
    const r = await verifyLicense()
    await refreshLic()
    if (r.state === 'verified') toastOk(t('license.verifyOk'))
    else if (r.state === 'revoked') toastErr(t('license.onlineRevoked'))
    else if (r.state === 'unbound') toastOk(t('license.unboundToast'))
    else if (r.error) toastErr(r.error)
  } catch (e) {
    toastErr(e.message)
  } finally {
    licBusy.value = false
  }
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

// ---------- 许可证状态判定(解绑 ≠ 吊销) ----------
// 已吊销:仅服务端 verify 判定 revoked 时显示"已吊销"(唯一允许显示吊销的场景)
const licRevoked = computed(
  () =>
    licOnline.value.verify_state === 'revoked' ||
    licOnline.value.sync_state === 'revoked' ||
    licOnline.value.state === 'revoked',
)
// 管理员强制解绑:SSE 事件 source=admin / verify unbound → 显示"请重新激活"
const licAdminUnbound = computed(
  () =>
    licOnline.value.unbind_source === 'admin' ||
    licOnline.value.unbind_reason === 'admin_unbound' ||
    (licOnline.value.verify_state === 'unbound' && licOnline.value.unbind_reason !== 'user_unbound'),
)

// 用户主动解绑(确认后只解 Binding:保留 Key,License 不会被吊销,可重新激活)
function confirmUnbind() {
  confirmAction({
    title: t('license.unbind'),
    message: t('license.unbindConfirmDesc'),
    okText: t('license.unbindConfirm'),
    danger: true,
  })
    .then(() => deactivate())
    .catch(() => {})
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

/* 横向 tab(仿 3x-ui antd Tabs:选中项底部一条主题色 ink 线,无滚动条) */
.h-tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid var(--dm-line);
  padding: 0 4px;
}
.h-tab {
  position: relative;
  padding: 10px 16px;
  border: none;
  background: transparent;
  color: var(--dm-muted);
  font-size: 13.5px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.2s;
}
.h-tab:hover {
  color: var(--dm-text);
}
.h-tab.active {
  color: var(--color-brand);
}
.h-tab.active::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  height: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--color-brand);
}

/* 保存后"需重启生效"警告(仿 3x-ui Alert type=warning:跟在重启面板按钮后面同一行) */
.restart-hint-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 8px;
  font-size: 12.5px;
  color: #fbbf24;
  background: rgba(245, 158, 11, 0.12);
  border: 1px solid rgba(245, 158, 11, 0.35);
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
  width: 420px;
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
  background: var(--color-brand);
  border-color: var(--color-brand);
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

/* 通知事件分组(仿 3x-ui NotificationGroup:分组卡片 + 标题 + 计数 + 事件行) */
.notify-groups {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-width: 460px;
}
.ng-group {
  border: 1px solid var(--dm-line);
  border-radius: 10px;
  background: var(--dm-surface2/40);
  padding: 10px 12px;
}
.ng-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--dm-text);
  margin-bottom: 6px;
}
.ng-title svg {
  color: var(--color-brand);
}
.ng-count {
  margin-left: auto;
  font-size: 11px;
  font-weight: 500;
  color: var(--dm-muted);
  background: var(--dm-surface2);
  border: 1px solid var(--dm-line);
  border-radius: 999px;
  padding: 1px 8px;
}
.ng-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 4px;
  border-radius: 6px;
  font-size: 12.5px;
  color: var(--dm-muted);
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, color 0.15s;
}
.ng-item:hover {
  background: var(--dm-surface2);
  color: var(--dm-text);
}
.ng-item input {
  accent-color: var(--color-brand);
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
  accent-color: var(--color-brand);
}

@media (max-width: 900px) {
  .settings-page {
    flex-direction: column;
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
