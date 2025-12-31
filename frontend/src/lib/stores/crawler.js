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
function createConfigStore() {
    const { subscribe, set, update } = writable({
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
        fetchMode: 'http',
        headless: true,
    });

    return {
        subscribe,
        update,
        reset: () => set({
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
            fetchMode: 'http',
            headless: true,
        }),
    };
}

export const configStore = createConfigStore();
