### Blog Post Series: Understanding Web Applications in oCIS

#### Part 1: Overview and How to Load Extensions

---

**Introduction to Web Applications in oCIS**

In today's fast-paced digital world, web applications play a crucial role in enhancing user experience and functionality.
Infinite Scale comes with a world-class web interface to manage file resources, but it can be extended by utilizing oCIS as a construction set for custom web apps.

For organizations using ownCloud Infinite Scale (oCIS), understanding how to leverage web applications can significantly enhance their productivity and user engagement.

In this series, we will delve into the world of web applications in oCIS, focusing on how extensions are loaded and utilized.

---

**Use Cases and Benefits of Custom Extensions**

The ability to provide custom extensions in oCIS opens up a myriad of possibilities for organizations. Here are some key use cases and benefits:

1. **Tailored User Experience**:\
   Custom extensions allow organizations to create a unique user experience that aligns with their specific needs. For instance, a company can develop a custom dashboard that displays relevant metrics and reports, enhancing productivity and decision-making.

2. **Third-Party Integrations**:\
   Web applications enable seamless integration with third-party services and tools, enhancing overall functionality. Organizations can integrate CRM systems, marketing automation tools, or custom data visualization tools directly into their oCIS environment, providing a seamless workflow for users.

3. **Enhanced Security and Compliance**:\
   Custom extensions can help organizations adhere to specific security and compliance requirements by adding features like custom authentication mechanisms, data encryption tools, or compliance reporting modules.

4. **Branding and Identity**:\
   By customizing the look and feel of the web applications, organizations can ensure their brand identity is consistently represented across their digital platforms. This can include custom themes, logos, and color schemes.

5. **Innovative Features**:\
   Custom extensions allow organizations to experiment with new features and functionalities that are not available in the default setup. This can include AI-powered tools, advanced analytics, or unique collaboration features.

The ability to provide custom extensions makes oCIS a powerful and flexible platform that can adapt to the evolving needs of any organization.
It empowers operators of the cloud to craft solutions that are not only functional but also aligned with their strategic goals.

---

**Loading Extensions in oCIS**

In oCIS, extensions can be loaded at both build time and runtime. Understanding the difference between these two methods is key to effectively managing and utilizing extensions.

- **Build Time Extensions**:\
  These are integrated into the binary during the build process. They are part of the core system and cannot be altered without rebuilding the system. This ensures a stable and consistent environment where critical applications are always available.

- **Run Time Extensions**:\
  These are loaded dynamically at runtime, providing greater flexibility. They can be placed in a designated directory and are automatically picked up by the system. This allows you to easily add, update, or remove extensions as needed, without the need for a system rebuild.

Extensions, also known as apps, are written in JavaScript, but we use TypeScript because of all its benefits.
TypeScript enhances the development process with features such as static typing, which helps catch errors early and improves code maintainability and scalability.

---

**How to Load Extensions**

1. **Build Time Extensions**:
   - Located in `<ocis_repo>/services/web/assets/apps`.
   - Integrated into the system during the build process.
   - These extensions are part of the binary and cannot be modified at runtime.

2. **Run Time Extensions**:
   - Stored in the directory specified by the `WEB_ASSET_APPS_PATH` environment variable.
   - By default, this path is `$OCIS_BASE_DATA_PATH/web/apps`, but it can be customized.
   - Run time extensions are automatically loaded from this directory, making it easy to add or remove extensions without rebuilding the system.

The ability to load extensions at runtime is particularly powerful, as it allows for a high degree of customization and flexibility.
You can quickly respond to changing needs by adding new functionality or removing outdated extensions.

---

**Manifest File**

Each web application must include a `manifest.json` or `manifest.yaml` file. This file contains essential information about the application, including its entry point and configuration details.

**Example of a manifest.json file**:
```json
{
  "entrypoint": "index.js",
  "config": {
    "maxWidth": 1280,
    "maxHeight": 1280
  }
}
```

The manifest file ensures that the system correctly recognizes and integrates the extension. It is a crucial component for defining how the web application should be loaded and what configurations it requires.

---

**Custom Configuration and Overwriting Options**

You can provide custom configurations in the `$OCIS_BASE_DATA_PATH/config/apps.yaml` file. This allows for fine-tuning of each extension's behavior and settings.

The `apps.yaml` file can contain custom settings that overwrite the default configurations specified in the extension's `manifest.json` file.

**Example of apps.yaml file**:
```yaml
image-viewer-obj:
  config:
    maxHeight: 640
    maxSize: 512
```

In this example, the `maxHeight` value specified in the `apps.yaml` file will overwrite the value from the `manifest.json` file.

This provides you with the flexibility to customize extensions to better meet the specific needs of their environment.

---

**Using Custom Assets**

Besides configuration, you might need to customize certain assets within an extension, such as logos or images.

This can be achieved by placing the custom assets in the path defined by `WEB_ASSET_APPS_PATH`.

For instance, if the default `image-viewer-dfx` application includes a logo that an organization wants to replace,
they can place the new logo in a directory structured like `WEB_ASSET_APPS_PATH/image-viewer-dfx/logo.png`.

The system will load this custom asset, replacing the default one. This method allows for easy and effective customization without altering the core application code.

---

**Configuration Example**

To illustrate how custom configurations and assets work together, consider the following scenario:

1. **Default Configuration**:
   ```json
   {
     "entrypoint": "index.js",
     "config": {
       "maxWidth": 1280,
       "maxHeight": 1280
     }
   }
   ```

2. **Custom Configuration in apps.yaml**:
   ```yaml
   image-viewer-obj:
     config:
       maxHeight: 640
       maxSize: 512
   ```

3. **Final Merged Configuration**:
   ```json
   {
     "external_apps": [
       {
         "id": "image-viewer-obj",
         "path": "index.js",
         "config": {
           "maxWidth": 1280,
           "maxHeight": 640,
           "maxSize": 512
         }
       }
     ]
   }
   ```

This example demonstrates how the system merges default and custom configurations to create the final settings used by the application.

---

**Conclusion**

In this first part of our series, we've covered the basics of web applications in oCIS, focusing on the importance of web applications,
how extensions are loaded, and how you can customize these extensions through configuration and asset overwriting.

Understanding these fundamentals is crucial for effectively managing and leveraging web applications in oCIS.

In the next post, we will dive deeper into the process of writing and running a basic extension.

Stay tuned for detailed instructions and tips on getting started with your first web extension in oCIS.

---

**Resources**:

- [Web Readme](https://github.com/owncloud/ocis/tree/master/services/web)
- [Overview of Available Extensions](https://github.com/owncloud/awesome-ocis)
- [Introduction PR](https://github.com/owncloud/ocis/pull/8523)
- [Design Document and Requirements](https://github.com/owncloud/ocis/issues/8392)
