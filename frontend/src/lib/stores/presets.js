import { writable } from 'svelte/store';

/**
 * Presets store for managing saved configuration presets
 */
function createPresetsStore() {
    const { subscribe, set, update } = writable({
        presets: [],       // List of PresetInfo objects
        loading: false,    // Loading state
        error: null,       // Error message if any
    });

    return {
        subscribe,

        /**
         * Load the list of available presets from the backend
         */
        async loadPresets() {
            update(state => ({ ...state, loading: true, error: null }));
            try {
                if (window.go && window.go.app && window.go.app.App) {
                    const presets = await window.go.app.App.ListPresets();
                    update(state => ({
                        ...state,
                        presets: presets || [],
                        loading: false,
                    }));
                }
            } catch (err) {
                update(state => ({
                    ...state,
                    loading: false,
                    error: err.message || 'Failed to load presets',
                }));
            }
        },

        /**
         * Save a new preset with the given name and config
         * @param {string} name - Preset name
         * @param {object} config - Configuration to save
         */
        async savePreset(name, config) {
            try {
                if (window.go && window.go.app && window.go.app.App) {
                    await window.go.app.App.SavePreset(name, config);
                    // Reload presets list after saving
                    await this.loadPresets();
                    return { success: true };
                }
                return { success: false, error: 'Backend not available' };
            } catch (err) {
                return { success: false, error: err.message || 'Failed to save preset' };
            }
        },

        /**
         * Load a preset by name and return its configuration
         * @param {string} name - Preset name
         */
        async loadPreset(name) {
            try {
                if (window.go && window.go.app && window.go.app.App) {
                    const config = await window.go.app.App.LoadPreset(name);
                    return { success: true, config };
                }
                return { success: false, error: 'Backend not available' };
            } catch (err) {
                return { success: false, error: err.message || 'Failed to load preset' };
            }
        },

        /**
         * Delete a preset by name
         * @param {string} name - Preset name
         */
        async deletePreset(name) {
            try {
                if (window.go && window.go.app && window.go.app.App) {
                    await window.go.app.App.DeletePreset(name);
                    // Reload presets list after deleting
                    await this.loadPresets();
                    return { success: true };
                }
                return { success: false, error: 'Backend not available' };
            } catch (err) {
                return { success: false, error: err.message || 'Failed to delete preset' };
            }
        },

        /**
         * Clear any error state
         */
        clearError() {
            update(state => ({ ...state, error: null }));
        },
    };
}

export const presetsStore = createPresetsStore();
