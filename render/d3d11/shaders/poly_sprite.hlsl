cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_atlas_size; // width, height of atlas texture in pixels
};

Texture2D<float4> t_atlas : register(t0);
SamplerState s_atlas : register(s0);

struct VSInput
{
    uint vertex_id : SV_VertexID;
    uint instance_id : SV_InstanceID;
    float4 bounds : INST_BOUNDS;           // min_x, min_y, max_x, max_y
    float4 content_mask : INST_MASK;       // min_x, min_y, max_x, max_y
    float4 tile_bounds : INST_TILE;        // min_x, min_y, max_x, max_y in atlas device pixels
    float4 corner_radii : INST_RADII;      // top_left, top_right, bottom_right, bottom_left
    float opacity : INST_OPACITY;
    uint grayscale : INST_GRAYSCALE;       // 1 = grayscale, 0 = full colour
    float2 pad0 : INST_PAD0;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float2 uv : TEXCOORD0;
    float4 bounds : BOUNDS;
    float4 content_mask : CONTENT_MASK;
    float4 corner_radii : CORNER_RADII;
    float opacity : OPACITY;
    uint grayscale : GRAYSCALE;
};

static const float2 QUAD_CORNERS[6] = {
    float2(0.0, 0.0),
    float2(1.0, 0.0),
    float2(0.0, 1.0),
    float2(0.0, 1.0),
    float2(1.0, 0.0),
    float2(1.0, 1.0)
};

PSInput vs_main(VSInput input)
{
    PSInput output;
    float2 corner = QUAD_CORNERS[input.vertex_id % 6];
    float2 pos = float2(
        lerp(input.bounds.x, input.bounds.z, corner.x),
        lerp(input.bounds.y, input.bounds.w, corner.y)
    );

    float2 ndc = float2(
        (pos.x / u_viewport_size.x) * 2.0 - 1.0,
        1.0 - (pos.y / u_viewport_size.y) * 2.0
    );

    float2 uv = float2(
        lerp(input.tile_bounds.x, input.tile_bounds.z, corner.x) / u_atlas_size.x,
        lerp(input.tile_bounds.y, input.tile_bounds.w, corner.y) / u_atlas_size.y
    );

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = pos;
    output.uv = uv;
    output.bounds = input.bounds;
    output.content_mask = input.content_mask;
    output.corner_radii = input.corner_radii;
    output.opacity = input.opacity;
    output.grayscale = input.grayscale;
    return output;
}

float rounded_box_sdf(float2 p, float2 half_size, float4 radii)
{
    float r = (p.x < 0.0) ? ((p.y < 0.0) ? radii.x : radii.w) : ((p.y < 0.0) ? radii.y : radii.z);
    r = min(r, min(half_size.x, half_size.y));
    float2 q = abs(p) - half_size + float2(r, r);
    return length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - r;
}

float4 ps_main(PSInput input) : SV_TARGET
{
    // Content mask clip
    if (input.content_mask.z > input.content_mask.x && input.content_mask.w > input.content_mask.y)
    {
        if (input.screen_pos.x < input.content_mask.x || input.screen_pos.x > input.content_mask.z ||
            input.screen_pos.y < input.content_mask.y || input.screen_pos.y > input.content_mask.w)
        {
            discard;
        }
    }

    float4 color = t_atlas.Sample(s_atlas, input.uv);

    if (input.grayscale != 0)
    {
        float gray = dot(color.rgb, float3(0.299, 0.587, 0.114));
        color.rgb = float3(gray, gray, gray);
    }

    color *= input.opacity;

    // Corner radii clip
    float max_radius = max(max(input.corner_radii.x, input.corner_radii.y),
                           max(input.corner_radii.z, input.corner_radii.w));
    if (max_radius > 0.0)
    {
        float2 size = float2(input.bounds.z - input.bounds.x, input.bounds.w - input.bounds.y);
        float2 half_size = size * 0.5;
        float2 centre = float2(input.bounds.x + half_size.x, input.bounds.y + half_size.y);
        float dist = rounded_box_sdf(input.screen_pos - centre, half_size, input.corner_radii);
        float alpha = saturate(0.5 - dist);
        color *= alpha;
    }

    return color;
}
