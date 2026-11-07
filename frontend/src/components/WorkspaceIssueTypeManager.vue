<template>
  <div class="w-full">
    <!-- 头部 -->
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-lg font-semibold text-gray-900">{{ t('workspaceIssueType.title') }}</h2>
        <p class="text-sm text-gray-500 mt-0.5">{{ t('workspaceIssueType.description') }}</p>
      </div>
      <button v-if="canManage" data-test="create-type-btn" @click="openCreateModal" class="create-btn">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('workspaceIssueType.createType') }}
      </button>
    </div>

    <!-- 类型列表 -->
    <div v-if="issueTypes.length > 0" class="space-y-2">
      <div
        v-for="type in issueTypes"
        :key="type.id"
        class="type-row"
        :class="{ 'is-default': type.is_default }"
      >
        <div class="type-row-main" @click="openEditModal(type)">
          <div class="type-icon" :style="{ backgroundColor: type.color }">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
              <path v-if="type.icon === 'circle'" stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              <path v-else-if="type.icon === 'square'" stroke-linecap="round" stroke-linejoin="round" d="M4 5h16v14H4z" />
              <path v-else-if="type.icon === 'bug'" stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              <path v-else-if="type.icon === 'task'" stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4M7.835 4.697a3.42 3.42 0 001.946-.806 3.42 3.42 0 014.438 0 3.42 3.42 0 001.946.806 3.42 3.42 0 013.138 3.138 3.42 3.42 0 00.806 1.946 3.42 3.42 0 010 4.438 3.42 3.42 0 00-.806 1.946 3.42 3.42 0 01-3.138 3.138 3.42 3.42 0 00-1.946.806 3.42 3.42 0 01-4.438 0 3.42 3.42 0 00-1.946-.806 3.42 3.42 0 01-3.138-3.138 3.42 3.42 0 00-.806-1.946 3.42 3.42 0 010-4.438 3.42 3.42 0 00.806-1.946 3.42 3.42 0 013.138-3.138z" />
              <path v-else-if="type.icon === 'check-square'" stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4M5 5h14a2 2 0 012 2v10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2z" />
              <path v-else-if="type.icon === 'bookmark'" stroke-linecap="round" stroke-linejoin="round" d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
              <path v-else-if="type.icon === 'flag'" stroke-linecap="round" stroke-linejoin="round" d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm0 0h7" />
              <path v-else-if="type.icon === 'star'" stroke-linecap="round" stroke-linejoin="round" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              <path v-else-if="type.icon === 'heart'" stroke-linecap="round" stroke-linejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
              <path v-else-if="type.icon === 'zap'" stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              <path v-else-if="type.icon === 'layers'" stroke-linecap="round" stroke-linejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2H7a2 2 0 00-2 2v2m10-9v6m-3-3h6" />
              <path v-else-if="type.icon === 'box'" stroke-linecap="round" stroke-linejoin="round" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              <path v-else-if="type.icon === 'database'" stroke-linecap="round" stroke-linejoin="round" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
              <path v-else-if="type.icon === 'file'" stroke-linecap="round" stroke-linejoin="round" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
              <path v-else-if="type.icon === 'code'" stroke-linecap="round" stroke-linejoin="round" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
              <path v-else-if="type.icon === 'terminal'" stroke-linecap="round" stroke-linejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              <path v-else-if="type.icon === 'settings'" stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path v-else-if="type.icon === 'users'" stroke-linecap="round" stroke-linejoin="round" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              <path v-else-if="type.icon === 'calendar'" stroke-linecap="round" stroke-linejoin="round" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              <path v-else-if="type.icon === 'clock'" stroke-linecap="round" stroke-linejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              <path v-else stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex items-center flex-wrap gap-x-2 gap-y-1">
              <span class="font-medium text-gray-900">{{ type.name }}</span>
              <span class="badge-level">L{{ type.level || 0 }}</span>
              <span v-if="type.is_default" class="badge-default">{{ t('workspaceIssueType.defaultBadge') }}</span>
              <span v-if="!type.is_active" class="badge-inactive">{{ t('workspaceIssueType.disabledBadge') }}</span>
              <span v-if="type.parent_type_id" class="badge-parent">{{ t('workspaceIssueType.parentConstraint') }}</span>
              <span v-if="type.allowed_child_type_ids?.length" class="badge-child">{{ t('workspaceIssueType.childConstraint') }}</span>
            </div>
            <p v-if="type.description" class="text-xs text-gray-500 mt-0.5 truncate">{{ type.description }}</p>
          </div>
        </div>
        <div v-if="canManage" class="type-row-actions">
          <button
            v-if="!type.is_default"
            @click.stop="toggleActive(type)"
            class="icon-action"
            :title="type.is_active ? t('workspaceIssueType.disableAction') : t('workspaceIssueType.enableAction')"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path v-if="type.is_active" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
              <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </button>
          <button
            v-if="!type.is_default"
            @click.stop="confirmDelete(type)"
            class="icon-action icon-action-danger"
            :title="t('workspaceIssueType.deleteAction')"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <div v-else class="empty-state">
      <svg class="w-12 h-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
      <p class="mt-2 text-sm text-gray-500">{{ t('workspaceIssueType.emptyText') }}</p>
    </div>

    <!-- 内嵌编辑抽屉 -->
    <Transition name="slide-fade">
      <div v-if="showEditDrawer" class="edit-drawer-overlay" @click.self="closeDrawer">
        <div class="edit-drawer">
          <div class="drawer-header">
            <div>
              <h3 class="drawer-title">
                {{ isCreating ? t('workspaceIssueType.createTitle') : t('workspaceIssueType.editTitle') + (selectedType?.name || '') }}
              </h3>
              <p class="drawer-subtitle">{{ t('workspaceIssueType.drawerSubtitle') }}</p>
            </div>
            <button @click="closeDrawer" class="close-btn">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="drawer-body">
            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.preview') }}</label>
              <div class="type-preview" :class="{ 'is-default': formData.is_default }">
                <div class="type-row">
                  <div class="type-row-main" style="cursor: default">
                    <div class="type-icon" :style="{ backgroundColor: formData.color }">
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                        <path v-if="formData.icon === 'circle'" stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        <path v-else-if="formData.icon === 'bug'" stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        <path v-else stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                      </svg>
                    </div>
                    <div class="flex-1 min-w-0">
                      <span class="font-medium text-gray-900">{{ formData.name || t('workspaceIssueType.unnamedType') }}</span>
                      <div class="flex flex-wrap gap-1 mt-1">
                        <span class="badge-level">L{{ formData.level || 0 }}</span>
                        <span v-if="formData.parent_type_id" class="badge-parent">{{ t('workspaceIssueType.parentConstraint') }}</span>
                        <span v-if="(formData.allowed_child_type_ids || []).length" class="badge-child">{{ t('workspaceIssueType.childConstraint') }}</span>
                        <span v-if="formData.is_default" class="badge-default">{{ t('workspaceIssueType.defaultBadge') }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.name') }} <span class="required">*</span></label>
              <input v-model="formData.name" type="text" class="form-input" :placeholder="t('workspaceIssueType.namePlaceholder')" />
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.icon') }}</label>
              <div class="icon-grid">
                <button
                  v-for="icon in ISSUE_TYPE_ICONS"
                  :key="icon"
                  class="icon-btn"
                  :class="{ active: formData.icon === icon }"
                  @click="formData.icon = icon"
                  :title="getIconName(icon)"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2">
                    <path v-if="icon === 'circle'" stroke-linecap="round" stroke-linejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    <path v-else-if="icon === 'bug'" stroke-linecap="round" stroke-linejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    <path v-else-if="icon === 'task'" stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4" />
                    <path v-else-if="icon === 'check-square'" stroke-linecap="round" stroke-linejoin="round" d="M9 12l2 2 4-4" />
                    <path v-else-if="icon === 'star'" stroke-linecap="round" stroke-linejoin="round" d="M12 2l3 7h7l-5.5 4 2 7L12 16l-6.5 4 2-7L2 9h7z" />
                    <path v-else-if="icon === 'flag'" stroke-linecap="round" stroke-linejoin="round" d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z" />
                    <path v-else stroke-linecap="round" stroke-linejoin="round" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                </button>
              </div>
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.color') }}</label>
              <div class="color-grid">
                <button
                  v-for="color in ISSUE_TYPE_COLORS"
                  :key="color"
                  class="color-btn"
                  :class="{ active: formData.color === color }"
                  :style="{ backgroundColor: color }"
                  @click="formData.color = color"
                />
              </div>
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.descriptionLabel') }}</label>
              <input v-model="formData.description" type="text" class="form-input" :placeholder="t('workspaceIssueType.descriptionPlaceholder')" />
            </div>

            <div class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.level') }}</label>
              <select v-model="formData.level" class="form-input">
                <option :value="0">{{ t('workspaceIssueType.level0') }}</option>
                <option :value="1">{{ t('workspaceIssueType.level1') }}</option>
                <option :value="2">{{ t('workspaceIssueType.level2') }}</option>
                <option :value="3">{{ t('workspaceIssueType.level3') }}</option>
                <option :value="4">{{ t('workspaceIssueType.level4') }}</option>
                <option :value="5">{{ t('workspaceIssueType.level5') }}</option>
              </select>
            </div>

            <div class="form-group" v-if="(formData.level || 0) > 0">
              <label class="form-label">{{ t('workspaceIssueType.parentType') }}</label>
              <select v-model="formData.parent_type_id" class="form-input">
                <option :value="undefined">{{ t('workspaceIssueType.noConstraint') }}</option>
                <option v-for="pt in parentTypeOptions" :key="pt.id" :value="pt.id">
                  {{ pt.name }} (L{{ pt.level }})
                </option>
              </select>
              <p class="text-xs text-gray-500 mt-1">{{ t('workspaceIssueType.parentHint') }}</p>
            </div>

            <div v-if="(formData.level ?? 0) < 5" class="form-group">
              <label class="form-label">{{ t('workspaceIssueType.allowedChildTypes') }}</label>
              <p class="text-xs text-gray-500 mb-2">{{ t('workspaceIssueType.allowedChildHint') }}</p>
              <div v-if="childTypeOptions.length === 0" class="text-xs text-gray-400 border border-dashed border-gray-300 rounded-lg p-3 text-center">
                {{ t('workspaceIssueType.allowedChildNoOptions') }}
              </div>
              <div v-else class="max-h-40 overflow-y-auto space-y-2 border border-gray-200 rounded-lg p-2 bg-gray-50">
                <label
                  v-for="ct in childTypeOptions"
                  :key="ct.id"
                  class="flex items-center space-x-2 text-sm text-gray-700 cursor-pointer py-0.5"
                >
                  <input
                    type="checkbox"
                    class="checkbox"
                    :checked="allowedChildIdSet.has(ct.id)"
                    @change="toggleAllowedChild(ct.id)"
                  />
                  <span>{{ ct.name }} <span class="text-gray-400 font-mono text-xs">(L{{ ct.level ?? 0 }})</span></span>
                </label>
              </div>
              <p v-if="(formData.allowed_child_type_ids || []).length === 0" class="text-xs text-gray-500 mt-1">{{ t('workspaceIssueType.allowedChildEmpty') }}</p>
            </div>

            <div class="form-group">
              <label class="checkbox-label">
                <input v-model="formData.is_default" type="checkbox" class="checkbox" />
                <span>{{ t('workspaceIssueType.setDefaultType') }}</span>
              </label>
              <p class="text-xs text-gray-500 mt-1">{{ t('workspaceIssueType.defaultHint') }}</p>
            </div>

            <!-- Custom Properties Binding (edit mode only) -->
            <div v-if="!isCreating && selectedType" class="form-group border-t border-gray-200 pt-4 mt-4">
              <div class="flex items-center justify-between mb-3">
                <label class="form-label mb-0">{{ t('workspaceIssueType.customProperties') }}</label>
                <button
                  v-if="canManage"
                  @click="showFieldBindModal = true"
                  class="text-sm text-indigo-600 hover:text-indigo-700 font-medium"
                  :disabled="availableFields.length === 0"
                >
                  {{ t('workspaceIssueType.addProperty') }}
                </button>
              </div>
              <p class="text-xs text-gray-500 mb-3">{{ t('workspaceIssueType.customPropertiesDesc') }}</p>

              <!-- Bound fields list -->
              <div v-if="typeFields.length > 0" class="space-y-2">
                <div v-for="f in typeFields" :key="f.field_id || f.id"
                  class="flex items-center justify-between p-2 bg-gray-50 rounded-lg border border-gray-200">
                  <div class="flex items-center space-x-2">
                    <span class="text-xs font-mono bg-gray-200 px-1.5 py-0.5 rounded">{{ getFieldTypeLabel(f.field_type || f.type) }}</span>
                    <span class="text-sm text-gray-700">{{ f.name || f.field_name || '#' + (f.field_id || f.id) }}</span>
                    <span v-if="f.is_required" class="text-xs text-red-500">{{ t('workspaceIssueType.required') }}</span>
                  </div>
                  <button v-if="canManage" @click="handleRemoveField(f)" class="text-gray-400 hover:text-red-500 p-1">✕</button>
                </div>
              </div>
              <div v-else class="text-center py-4 text-gray-400 text-sm border border-dashed border-gray-300 rounded-lg">
                {{ t('workspaceIssueType.noProperties') }}
              </div>
            </div>
          </div>

          <div class="drawer-footer">
            <button @click="closeDrawer" class="btn btn-secondary">{{ t('common.cancel') }}</button>
            <button v-if="canManage" @click="submitForm" class="btn btn-primary" :disabled="!formData.name">
              {{ isCreating ? t('common.create') : t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>

    <!-- Field Bind Modal -->
    <div v-if="showFieldBindModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" @click.self="showFieldBindModal = false">
      <div class="bg-white rounded-xl p-6 w-full max-w-md max-h-[70vh] overflow-y-auto">
        <h3 class="text-lg font-semibold text-gray-900 mb-4">{{ t('workspaceIssueType.addPropertiesTitle') }} "{{ selectedType?.name }}"</h3>
        <p class="text-sm text-gray-500 mb-4">{{ t('workspaceIssueType.addPropertiesDesc') }}</p>
        <div v-if="availableFields.length > 0" class="space-y-2">
          <div v-for="field in availableFields" :key="field.id"
            @click="handleAddField(field); showFieldBindModal = false"
            class="flex items-center justify-between p-3 border border-gray-200 rounded-lg hover:border-indigo-300 hover:bg-indigo-50 cursor-pointer transition">
            <div class="flex items-center space-x-3">
              <span class="text-xs font-mono bg-gray-100 px-1.5 py-0.5 rounded">{{ getFieldTypeLabel(field.field_type || field.type) }}</span>
              <span class="text-sm font-medium text-gray-800">{{ field.name }}</span>
            </div>
            <span class="text-indigo-500">+</span>
          </div>
        </div>
        <div v-else class="text-center py-8 text-gray-400 text-sm">
          {{ t('workspaceIssueType.allFieldsBound') }}
        </div>
        <div class="flex justify-end mt-4">
          <button @click="showFieldBindModal = false" class="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 text-sm">{{ t('common.close') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useI18n } from '@/composables/useI18n'
import type { IssueType, IssueTypeCreate, IssueTypeUpdate } from '@/types/issue-type'
import { ISSUE_TYPE_ICONS, ISSUE_TYPE_COLORS, getIconName } from '@/types/issue-type'
import { useConfirm } from '@/composables/useConfirm'
import * as issueTypeApi from '@/api/issue-type'
import * as customFieldApi from '@/api/custom-field'
import { useWorkspacePermissions } from '@/composables/usePermission'

const props = defineProps<{
  workspaceId: number
}>()

const { t } = useI18n()
const { confirm } = useConfirm()
const { can } = useWorkspacePermissions(() => props.workspaceId)
const canManage = computed(() => can('settings:manage'))

const issueTypes = ref<IssueType[]>([])
const loading = ref(false)
const showEditDrawer = ref(false)
const showFieldBindModal = ref(false)
const isCreating = ref(false)
const selectedType = ref<IssueType | null>(null)
const typeFields = ref<any[]>([])
const allCustomFields = ref<any[]>([])

const formData = ref<IssueTypeCreate & IssueTypeUpdate>({
  name: '',
  color: ISSUE_TYPE_COLORS[0],
  icon: ISSUE_TYPE_ICONS[0],
  description: '',
  level: 0,
  parent_type_id: undefined,
  allowed_child_type_ids: [],
  is_default: false,
  sequence: 1
})

const parentTypeOptions = computed(() => {
  const targetLevel = (formData.value.level || 0) - 1
  if (targetLevel < 0) return []
  const sid = selectedType.value?.id
  return issueTypes.value.filter(t => (t.level || 0) === targetLevel && t.id !== sid)
})

const childTypeOptions = computed(() => {
  const myLevel = formData.value.level ?? 0
  const sid = selectedType.value?.id
  return issueTypes.value.filter(t => {
    if (sid != null && t.id === sid) return false
    return (t.level ?? 0) > myLevel
  })
})

const allowedChildIdSet = computed(() => new Set(formData.value.allowed_child_type_ids || []))

function toggleAllowedChild(typeId: number) {
  const cur = [...(formData.value.allowed_child_type_ids || [])]
  const i = cur.indexOf(typeId)
  if (i >= 0) cur.splice(i, 1)
  else cur.push(typeId)
  formData.value.allowed_child_type_ids = cur
}

watch(
  () => [formData.value.level, selectedType.value?.id, issueTypes.value.map(t => t.id).join(',')] as const,
  () => {
    if ((formData.value.level || 0) === 0) formData.value.parent_type_id = undefined
    const validParent = new Set(parentTypeOptions.value.map(t => t.id))
    const pid = formData.value.parent_type_id
    if (pid != null && !validParent.has(pid)) formData.value.parent_type_id = undefined
    const validChild = new Set(childTypeOptions.value.map(t => t.id))
    const allowed = formData.value.allowed_child_type_ids || []
    const next = allowed.filter(id => validChild.has(id))
    if (next.length !== allowed.length) formData.value.allowed_child_type_ids = next
  }
)

const availableFields = computed(() => {
  const boundIds = new Set(typeFields.value.map((f: any) => f.field_id || f.id))
  return allCustomFields.value.filter((f: any) => !boundIds.has(f.id))
})

async function loadData() {
  if (!props.workspaceId) return
  loading.value = true
  try {
    issueTypes.value = await issueTypeApi.getIssueTypes(props.workspaceId)
    allCustomFields.value = await customFieldApi.listCustomFields(props.workspaceId)
  } catch (error) {
    console.error('Failed to load issue types:', error)
  } finally {
    loading.value = false
  }
}

async function loadTypeFields(typeId: number) {
  try {
    typeFields.value = await issueTypeApi.getIssueTypeFields(typeId)
  } catch { typeFields.value = [] }
}

function openCreateModal() {
  isCreating.value = true
  selectedType.value = null
  typeFields.value = []
  formData.value = {
    name: '',
    color: ISSUE_TYPE_COLORS[0],
    icon: ISSUE_TYPE_ICONS[0],
    description: '',
    level: 0,
    parent_type_id: undefined,
    allowed_child_type_ids: [],
    is_default: false,
    sequence: issueTypes.value.length + 1
  }
  showEditDrawer.value = true
}

async function openEditModal(type: IssueType) {
  isCreating.value = false
  selectedType.value = type
  formData.value = {
    name: type.name,
    color: type.color,
    icon: type.icon,
    description: type.description || '',
    level: type.level || 0,
    parent_type_id: type.parent_type_id || undefined,
    allowed_child_type_ids: type.allowed_child_type_ids ? [...type.allowed_child_type_ids] : [],
    is_default: type.is_default,
    sequence: type.sequence
  }
  await loadTypeFields(type.id)
  showEditDrawer.value = true
}

function closeDrawer() {
  showEditDrawer.value = false
  selectedType.value = null
  isCreating.value = false
}

function payloadForApi(): IssueTypeCreate & IssueTypeUpdate {
  const fd = formData.value
  const allowed = fd.allowed_child_type_ids || []
  return {
    ...fd,
    parent_type_id: (fd.level || 0) > 0 ? fd.parent_type_id : undefined,
    allowed_child_type_ids: [...allowed],
  }
}

async function submitForm() {
  if (!formData.value.name) return
  try {
    if (isCreating.value) {
      await issueTypeApi.createIssueType(props.workspaceId, payloadForApi())
    } else if (selectedType.value) {
      await issueTypeApi.updateIssueType(selectedType.value.id, payloadForApi())
    }
    closeDrawer()
    await loadData()
  } catch (error) {
    console.error('Failed to submit form:', error)
  }
}

async function toggleActive(type: IssueType) {
  const actionKey = type.is_active ? 'workspaceIssueType.disableAction' : 'workspaceIssueType.enableAction'
  if (await confirm({
    title: t(type.is_active ? 'workspaceIssueType.disableConfirmTitle' : 'workspaceIssueType.enableConfirmTitle'),
    message: t('workspaceIssueType.confirmToggleMessage', { action: t(actionKey), name: type.name }),
    confirmText: t(actionKey),
    danger: type.is_active,
  })) {
    try {
      await issueTypeApi.disableIssueType(type.id, !type.is_active)
      await loadData()
    } catch (error) {
      console.error('Failed to toggle active:', error)
    }
  }
}

async function confirmDelete(type: IssueType) {
  if (await confirm({
    title: t('workspaceIssueType.deleteConfirmTitle'),
    message: t('workspaceIssueType.deleteConfirmMessage', { name: type.name }),
    danger: true,
    confirmText: t('workspaceIssueType.deleteConfirmBtn')
  })) {
    try {
      await issueTypeApi.deleteIssueType(type.id)
      await loadData()
    } catch (error) {
      console.error('Failed to delete type:', error)
    }
  }
}

// ===== Field binding =====
async function handleAddField(field: any) {
  if (!selectedType.value) return
  try {
    await issueTypeApi.addFieldToIssueType(selectedType.value.id, {
      field_id: field.id,
      is_required: false,
      sequence: typeFields.value.length + 1
    })
    await loadTypeFields(selectedType.value.id)
    await loadData()
  } catch (e) { console.error('Failed to add field:', e) }
}
async function handleRemoveField(field: any) {
  if (!selectedType.value) return
  try {
    const fieldId = field.field_id || field.id
    await issueTypeApi.removeFieldFromIssueType(selectedType.value.id, fieldId)
    await loadTypeFields(selectedType.value.id)
  } catch (e) { console.error('Failed to remove field:', e) }
}
function getFieldTypeLabel(type: string): string {
  const map: Record<string, string> = {
    text: t('common.text'), number: t('common.number'), dropdown: t('common.dropdown'),
    boolean: t('common.boolean'), date: t('common.date'), member: t('common.member'), url: t('common.url')
  }
  return map[type] || type
}

defineExpose({ loadData })

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.create-btn {
  @apply flex items-center space-x-1 px-3 py-1.5 text-sm bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition;
}

.type-row {
  @apply flex items-center justify-between px-3 py-2 bg-white border border-gray-200 rounded-lg hover:border-indigo-300 transition;
}

.type-row.is-default {
  @apply border-indigo-200 bg-indigo-50;
}

.type-row-main {
  @apply flex items-center space-x-3 flex-1 min-w-0 cursor-pointer;
}

.type-icon {
  @apply w-8 h-8 rounded-md flex items-center justify-center text-white shrink-0 overflow-hidden;
}

.type-row-actions {
  @apply flex items-center space-x-1;
}

.icon-action {
  @apply p-1.5 rounded hover:bg-gray-100 text-gray-500 hover:text-gray-700 transition;
}

.icon-action-danger {
  @apply hover:bg-red-50 hover:text-red-600;
}

.badge-default {
  @apply px-1.5 py-0.5 text-xs rounded bg-indigo-100 text-indigo-700;
}

.badge-inactive {
  @apply px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-600;
}

.badge-level {
  @apply px-1.5 py-0.5 text-xs rounded bg-blue-100 text-blue-700 font-mono;
}

.badge-parent {
  @apply px-1.5 py-0.5 text-xs rounded bg-yellow-100 text-yellow-700;
}

.badge-child {
  @apply px-1.5 py-0.5 text-xs rounded bg-purple-100 text-purple-700;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-12 text-center;
}

/* 编辑抽屉 */
.edit-drawer-overlay {
  @apply fixed inset-0 bg-black bg-opacity-30 z-40 flex justify-end;
}

.edit-drawer {
  @apply bg-white w-full max-w-md h-full shadow-xl flex flex-col;
}

.drawer-header {
  @apply flex items-center justify-between px-6 py-4 border-b border-gray-200 shrink-0;
}

.drawer-title {
  @apply text-lg font-semibold text-gray-900;
}

.drawer-subtitle {
  @apply text-sm text-gray-500 mt-0.5;
}

.close-btn {
  @apply p-2 rounded-lg hover:bg-gray-100 text-gray-500 transition;
}

.drawer-body {
  @apply px-6 py-4 flex-1 overflow-y-auto;
}

.drawer-footer {
  @apply flex items-center justify-end space-x-3 px-6 py-4 border-t border-gray-200 shrink-0;
}

.type-preview {
  @apply bg-white rounded-lg border border-gray-200 p-3;
}

.type-preview.is-default {
  @apply border-indigo-200 bg-indigo-50;
}

/* 表单 */
.form-group {
  @apply mb-4;
}

.form-label {
  @apply block text-sm font-medium text-gray-700 mb-1;
}

.required {
  @apply text-red-500;
}

.form-input {
  @apply w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500;
}

.icon-grid {
  @apply grid grid-cols-7 gap-2;
}

.icon-btn {
  @apply w-9 h-9 p-1.5 rounded-lg border border-gray-200 hover:border-indigo-300 hover:bg-indigo-50 transition flex items-center justify-center;
}

.icon-btn.active {
  @apply border-indigo-500 bg-indigo-100;
}

.color-grid {
  @apply flex flex-wrap gap-2;
}

.color-btn {
  @apply w-7 h-7 rounded-full border-2 border-transparent hover:border-gray-400 transition;
}

.color-btn.active {
  @apply border-gray-900;
}

.checkbox-label {
  @apply flex items-center space-x-2 cursor-pointer;
}

.checkbox {
  @apply w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500;
}

.btn {
  @apply px-4 py-2 rounded-lg font-medium transition disabled:opacity-50 disabled:cursor-not-allowed;
}

.btn-primary {
  @apply bg-indigo-600 text-white hover:bg-indigo-700;
}

.btn-secondary {
  @apply bg-gray-100 text-gray-700 hover:bg-gray-200;
}

.slide-fade-enter-active,
.slide-fade-leave-active {
  transition: all 0.25s ease;
}

.slide-fade-enter-from .edit-drawer,
.slide-fade-leave-to .edit-drawer {
  transform: translateX(100%);
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  opacity: 0;
}
</style>
