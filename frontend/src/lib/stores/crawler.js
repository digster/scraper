import { writable } from 'svelte/store';

// Crawler state store
function createCrawlerStore() {
    const { subscribe, set, update } = writable({
        status: 'stopped', // stopped, running, paused
        progress: null,
        logs: [],
        error: null,
    });

    return {
        subscribe,
        setStatus: (status) => update(state => ({ ...state, status })),
        setProgress: (progress) => update(state => ({ ...state, progress })),
        addLog: (log) => update(state => ({
            ...state,
            logs: [...state.logs.slice(-499), log] // Keep last 500 logs
        })),
        setError: (error) => update(state => ({ ...state, error })),
        clearLogs: () => update(state => ({ ...state, logs: [] })),
        reset: () => set({
            status: 'stopped',
            progress: null,
            logs: [],
            error: null,
        }),
    };
}

export const crawlerStore = createCrawlerStore();

// Config store for form state
// Default config values - used for reset and as base for presets
const defaultConfig = {
    url: '',
    concurrent: false,
    delay: '1s',
    maxDepth: 10,
    outputDir: '',
    stateFile: '',
    prefixFilter: '',
    excludeExtensions: 'js,css,png,jpg,gif,svg,ico,woff,woff2,ttf,eot',
    linkSelectors: 'a[href]',
    verbose: false,
    userAgent: '',
    ignoreRobots: false,
    minContent: 100,
    disableReadability: false,
    fetchMode: 'http',
    headless: true,
    waitForLogin: false,
    // Pagination settings (browser mode only)
    enablePagination: false,
    paginationSelector: '',
    maxPaginationClicks: 100,
    paginationWait: '2s',
    paginationWaitSelector: '',
    paginationStopOnDuplicate: true,
    // Anti-bot settings (visible only in non-headless browser mode)
    hideWebdriver: false,
    spoofPlugins: false,
    spoofLanguages: false,
    spoofWebGL: false,
    addCanvasNoise: false,
    naturalMouseMovement: false,
    randomTypingDelays: false,
    naturalScrolling: false,
    randomActionDelays: false,
    randomClickOffset: false,
    rotateUserAgent: false,
    randomViewport: false,
    matchTimezone: false,
    timezone: '',
    // URL normalization settings
    normalizeUrls: true,
    lowercasePaths: false,
};

function createConfigStore() {
    const { subscribe, set, update } = writable({ ...defaultConfig });

    // Store for getting current value synchronously
    let currentValue = { ...defaultConfig };
    subscribe(value => { currentValue = value; });

    return {
        subscribe,
        update,

        /**
         * Get the current config object (useful for saving presets)
         * Excludes outputDir and stateFile as per spec
         */
        getPresetConfig: () => {
            const { outputDir, stateFile, ...presetFields } = currentValue;
            return presetFields;
        },

        /**
         * Apply a loaded preset config to the form
         * Preserves outputDir and stateFile from current config
         * @param {object} preset - The preset config to apply
         */
        applyPreset: (preset) => {
            update(current => ({
                ...defaultConfig,        // Start with defaults
                ...preset,               // Apply preset values
                outputDir: current.outputDir,    // Preserve job-specific paths
                stateFile: current.stateFile,
            }));
        },

        reset: () => set({ ...defaultConfig }),
    };
}

export const configStore = createConfigStore();
