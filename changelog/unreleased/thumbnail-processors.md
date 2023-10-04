Enhancement: Thumbnail generation with image processors

Thumbnails can now be changed during creation, previously the images were always scaled to fit the given frame,
but it could happen that the images were cut off because they could not be placed better due to the aspect ratio.

This pr introduces the possibility of specifying how the behavior should be, following processors are available

* resize
* fit
* fill
* thumbnail

the processor can be applied by adding the processor query param to the request, e.g. `processor=fit`, `processor=fill`, ...

to find out more how the individual processors work please read https://github.com/disintegration/imaging

if no processor is provided it behaves the same as before (resize for gif's and thumbnail for all other)

https://github.com/owncloud/ocis/pull/6992
