//kudos to: https://github.com/dgraph-io/dgraph-docs

function debounce(func, wait, immediate) {
    var timeout;

    return function () {
        var context = this,
            args = arguments;
        var later = function () {
            timeout = null;
            if (!immediate) func.apply(context, args);
        };

        var callNow = immediate && !timeout;
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
        if (callNow) func.apply(context, args);
    };
}

function createCookie(name, val, days) {
    var expires = "";
    if (days) {
        var date = new Date();
        date.setTime(date.getTime() + days * 24 * 60 * 60 * 1000);
        expires = "; expires=" + date.toUTCString();
    }

    document.cookie = name + "=" + val + expires + "; path=/";
}

function readCookie(name) {
    var nameEQ = name + "=";
    var ca = document.cookie.split(";");
    for (var i = 0; i < ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == " ") c = c.substring(1, c.length);
        if (c.indexOf(nameEQ) == 0) return c.substring(nameEQ.length, c.length);
    }
    return null;
}

function eraseCookie(name) {
    createCookie(name, "", -1);
}

/**
 * getCurrentProductAndVersion gets the current product and doc version from the URL path.
 *
 * @param pathname {String} - current path in a format like '/product/version/docs'.
 * @return {Object} - an object containing product and version, e.g., { product: 'gateway', version: 'v1.0' }
 */
function getCurrentProductAndVersion(pathname) {
    const pathParts = pathname.split("/");

    const product = pathParts[1]; // Assumes product is the first part after the root
    const version = pathParts[2]; // Assumes version is the second part after the product

    // Validate version format
    if (version === "master" || /v\d+\.\d+(\.\d+)?/.test(version) || version === "latest") {
        return { product, version };
    }

    // Default to empty version for "latest"
    return { product, version: "" };
}

// getPathBeforeVersionName gets the URL path up to the version prefix
function getPathBeforeVersionName(location, product) {
    return `/${product}/`;
}

// getPathAfterVersionName gets the URL path after the version prefix
function getPathAfterVersionName(location, version) {
    const pathParts = location.pathname.split("/");
    const startSliceIndex = version ? 3 : 2; // Adjust based on whether there's a version in the path
    const pathAfterVersion = pathParts.slice(startSliceIndex).join("/");

    return pathAfterVersion + location.hash;
}

(function () {
    // Get the current product and version
    const { product, version: currentVersion } = getCurrentProductAndVersion(location.pathname);

    // Version selector
    const versionSelectors = document.getElementsByClassName("version-selector");
    if (versionSelectors.length) {
        versionSelectors[0].addEventListener("change", function (e) {
            const targetVersion = e.target.value;

            if (currentVersion !== targetVersion) {
                const basePath = getPathBeforeVersionName(location, product);
                const currentPath = getPathAfterVersionName(location, currentVersion);

                // Include "latest" explicitly in the path if it’s the target version
                const targetPath = targetVersion
                    ? `${basePath}${targetVersion}/${currentPath}`
                    : `${basePath}latest/${currentPath}`;

                location.assign(targetPath);
            }
        });

        // Set the current version in the dropdown
        const versionSelector = versionSelectors[0];
        const options = versionSelector.options;
        for (let i = 0; i < options.length; i++) {
            if (options[i].value.replace(/\s\(latest\)/, "") === currentVersion) {
                options[i].selected = true;
                break;
            }
        }
    }

    // Open all external links in a new tab
    const links = document.links;
    for (let i = 0; i < links.length; i++) {
        if (links[i].hostname !== window.location.hostname) {
            links[i].target = "_blank";
        }
    }
})();