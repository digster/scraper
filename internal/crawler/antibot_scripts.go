package crawler

// JavaScript injection scripts for anti-bot bypass
// These scripts are injected before page load to modify browser fingerprint

// HideWebdriverScript removes the navigator.webdriver flag that identifies automated browsers
const HideWebdriverScript = `
Object.defineProperty(navigator, 'webdriver', {
    get: () => undefined,
});

// Also hide automation-related properties
Object.defineProperty(navigator, 'plugins', {
    get: () => {
        const original = Object.getOwnPropertyDescriptor(Navigator.prototype, 'plugins');
        if (original && original.get) {
            const plugins = original.get.call(navigator);
            if (plugins.length === 0) {
                return undefined;
            }
            return plugins;
        }
        return undefined;
    },
});
`

// SpoofPluginsScript injects realistic browser plugins to match a normal Chrome profile
const SpoofPluginsScript = `
Object.defineProperty(navigator, 'plugins', {
    get: () => {
        const pluginData = [
            {
                name: 'Chrome PDF Plugin',
                description: 'Portable Document Format',
                filename: 'internal-pdf-viewer',
                mimeTypes: [
                    { type: 'application/x-google-chrome-pdf', suffixes: 'pdf', description: 'Portable Document Format' }
                ]
            },
            {
                name: 'Chrome PDF Viewer',
                description: '',
                filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai',
                mimeTypes: [
                    { type: 'application/pdf', suffixes: 'pdf', description: '' }
                ]
            },
            {
                name: 'Native Client',
                description: '',
                filename: 'internal-nacl-plugin',
                mimeTypes: [
                    { type: 'application/x-nacl', suffixes: '', description: 'Native Client Executable' },
                    { type: 'application/x-pnacl', suffixes: '', description: 'Portable Native Client Executable' }
                ]
            }
        ];

        const plugins = {
            length: pluginData.length,
            item: (index) => plugins[index],
            namedItem: (name) => {
                for (let i = 0; i < pluginData.length; i++) {
                    if (plugins[i].name === name) return plugins[i];
                }
                return null;
            },
            refresh: () => {}
        };

        pluginData.forEach((data, index) => {
            const plugin = {
                name: data.name,
                description: data.description,
                filename: data.filename,
                length: data.mimeTypes.length
            };

            data.mimeTypes.forEach((mime, mimeIndex) => {
                plugin[mimeIndex] = {
                    type: mime.type,
                    suffixes: mime.suffixes,
                    description: mime.description,
                    enabledPlugin: plugin
                };
            });

            plugins[index] = plugin;
        });

        return plugins;
    },
});

// Also spoof mimeTypes
Object.defineProperty(navigator, 'mimeTypes', {
    get: () => {
        const mimeData = [
            { type: 'application/x-google-chrome-pdf', suffixes: 'pdf', description: 'Portable Document Format' },
            { type: 'application/pdf', suffixes: 'pdf', description: '' },
            { type: 'application/x-nacl', suffixes: '', description: 'Native Client Executable' },
            { type: 'application/x-pnacl', suffixes: '', description: 'Portable Native Client Executable' }
        ];

        const mimeTypes = {
            length: mimeData.length,
            item: (index) => mimeTypes[index],
            namedItem: (name) => {
                for (let i = 0; i < mimeData.length; i++) {
                    if (mimeTypes[i].type === name) return mimeTypes[i];
                }
                return null;
            }
        };

        mimeData.forEach((data, index) => {
            mimeTypes[index] = {
                type: data.type,
                suffixes: data.suffixes,
                description: data.description
            };
        });

        return mimeTypes;
    },
});
`

// SpoofLanguagesScript sets navigator.languages to common browser values
const SpoofLanguagesScript = `
Object.defineProperty(navigator, 'languages', {
    get: () => ['en-US', 'en'],
});

Object.defineProperty(navigator, 'language', {
    get: () => 'en-US',
});
`

// SpoofWebGLScript overrides WebGL vendor and renderer to avoid fingerprinting
const SpoofWebGLScript = `
const getParameterOriginal = WebGLRenderingContext.prototype.getParameter;
WebGLRenderingContext.prototype.getParameter = function(parameter) {
    // UNMASKED_VENDOR_WEBGL
    if (parameter === 37445) {
        return 'Intel Inc.';
    }
    // UNMASKED_RENDERER_WEBGL
    if (parameter === 37446) {
        return 'Intel Iris OpenGL Engine';
    }
    return getParameterOriginal.call(this, parameter);
};

// Also handle WebGL2
if (typeof WebGL2RenderingContext !== 'undefined') {
    const getParameter2Original = WebGL2RenderingContext.prototype.getParameter;
    WebGL2RenderingContext.prototype.getParameter = function(parameter) {
        // UNMASKED_VENDOR_WEBGL
        if (parameter === 37445) {
            return 'Intel Inc.';
        }
        // UNMASKED_RENDERER_WEBGL
        if (parameter === 37446) {
            return 'Intel Iris OpenGL Engine';
        }
        return getParameter2Original.call(this, parameter);
    };
}
`

// CanvasNoiseScript adds subtle noise to canvas fingerprinting attempts
const CanvasNoiseScript = `
const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
const originalToBlob = HTMLCanvasElement.prototype.toBlob;
const originalGetImageData = CanvasRenderingContext2D.prototype.getImageData;

// Add noise to a single pixel value
function addNoise(value) {
    const noise = Math.floor(Math.random() * 3) - 1; // -1, 0, or 1
    return Math.max(0, Math.min(255, value + noise));
}

// Override toDataURL
HTMLCanvasElement.prototype.toDataURL = function(type, quality) {
    const ctx = this.getContext('2d');
    if (ctx && this.width > 0 && this.height > 0) {
        try {
            const imageData = ctx.getImageData(0, 0, this.width, this.height);
            // Only modify a small percentage of pixels to avoid visible artifacts
            for (let i = 0; i < imageData.data.length; i += 4) {
                if (Math.random() < 0.01) { // 1% of pixels
                    imageData.data[i] = addNoise(imageData.data[i]);     // R
                    imageData.data[i + 1] = addNoise(imageData.data[i + 1]); // G
                    imageData.data[i + 2] = addNoise(imageData.data[i + 2]); // B
                }
            }
            ctx.putImageData(imageData, 0, 0);
        } catch (e) {
            // Canvas may be tainted, ignore
        }
    }
    return originalToDataURL.call(this, type, quality);
};

// Override toBlob
HTMLCanvasElement.prototype.toBlob = function(callback, type, quality) {
    const ctx = this.getContext('2d');
    if (ctx && this.width > 0 && this.height > 0) {
        try {
            const imageData = ctx.getImageData(0, 0, this.width, this.height);
            for (let i = 0; i < imageData.data.length; i += 4) {
                if (Math.random() < 0.01) {
                    imageData.data[i] = addNoise(imageData.data[i]);
                    imageData.data[i + 1] = addNoise(imageData.data[i + 1]);
                    imageData.data[i + 2] = addNoise(imageData.data[i + 2]);
                }
            }
            ctx.putImageData(imageData, 0, 0);
        } catch (e) {
            // Canvas may be tainted, ignore
        }
    }
    return originalToBlob.call(this, callback, type, quality);
};

// Override getImageData to add noise on read
CanvasRenderingContext2D.prototype.getImageData = function(sx, sy, sw, sh) {
    const imageData = originalGetImageData.call(this, sx, sy, sw, sh);
    for (let i = 0; i < imageData.data.length; i += 4) {
        if (Math.random() < 0.01) {
            imageData.data[i] = addNoise(imageData.data[i]);
            imageData.data[i + 1] = addNoise(imageData.data[i + 1]);
            imageData.data[i + 2] = addNoise(imageData.data[i + 2]);
        }
    }
    return imageData;
};
`

// BuildInjectionScripts returns all scripts to inject based on config
func BuildInjectionScripts(config AntiBotConfig) []string {
	var scripts []string

	if config.HideWebdriver {
		scripts = append(scripts, HideWebdriverScript)
	}
	if config.SpoofPlugins {
		scripts = append(scripts, SpoofPluginsScript)
	}
	if config.SpoofLanguages {
		scripts = append(scripts, SpoofLanguagesScript)
	}
	if config.SpoofWebGL {
		scripts = append(scripts, SpoofWebGLScript)
	}
	if config.AddCanvasNoise {
		scripts = append(scripts, CanvasNoiseScript)
	}

	return scripts
}
