cbuffer GlobalConstants : register(b0)
{
    float2 u_viewport_size;
    float2 u_pad;
};

struct VSInput
{
    uint vertex_id : SV_VertexID;
    uint instance_id : SV_InstanceID;
    float4 bounds : INST_BOUNDS;           // min_x, min_y, max_x, max_y
    float4 corner_radii : INST_RADII;      // top_left, top_right, bottom_right, bottom_left
    float4 content_mask : INST_MASK;       // min_x, min_y, max_x, max_y
    float4 color : INST_COLOR;             // premultiplied rgba
    float4 elem_bounds : INST_ELEM_BOUNDS; // min_x, min_y, max_x, max_y
    float4 elem_radii : INST_ELEM_RADII;   // top_left, top_right, bottom_right, bottom_left
    float blur_radius : INST_BLUR;
    uint inset : INST_INSET;               // 0 = drop shadow, 1 = inset shadow
    uint pad0 : INST_PAD0;
    uint pad1 : INST_PAD1;
};

struct PSInput
{
    float4 position : SV_POSITION;
    float2 screen_pos : SCREEN_POS;
    float4 bounds : BOUNDS;
    float4 corner_radii : CORNER_RADII;
    float4 content_mask : CONTENT_MASK;
    float4 color : COLOR;
    float4 elem_bounds : ELEM_BOUNDS;
    float4 elem_radii : ELEM_ELEM_RADII;
    float blur_radius : BLUR_RADIUS;
    uint inset : INSET;
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

    output.position = float4(ndc, 0.0, 1.0);
    output.screen_pos = pos;
    output.bounds = input.bounds;
    output.corner_radii = input.corner_radii;
    output.content_mask = input.content_mask;
    output.color = input.color;
    output.elem_bounds = input.elem_bounds;
    output.elem_radii = input.elem_radii;
    output.blur_radius = input.blur_radius;
    output.inset = input.inset;
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

    float2 size = float2(input.bounds.z - input.bounds.x, input.bounds.w - input.bounds.y);
    float2 half_size = size * 0.5;
    float2 centre = float2(input.bounds.x + half_size.x, input.bounds.y + half_size.y);
    float2 p = input.screen_pos - centre;

    float dist = rounded_box_sdf(p, half_size, input.corner_radii);
    float sigma = max(0.5, input.blur_radius * 0.5);
    float alpha = saturate(1.0 - smoothstep(-sigma * 2.0, sigma * 2.0, dist));

    // Element mask for drop vs inset shadow
    float2 elem_size = float2(input.elem_bounds.z - input.elem_bounds.x, input.elem_bounds.w - input.elem_bounds.y);
    if (elem_size.x > 0.0 && elem_size.y > 0.0)
    {
        float2 elem_half = elem_size * 0.5;
        float2 elem_centre = float2(input.elem_bounds.x + elem_half.x, input.elem_bounds.y + elem_half.y);
        float elem_dist = rounded_box_sdf(input.screen_pos - elem_centre, elem_half, input.elem_radii);
        float elem_inside = saturate(0.5 - elem_dist);

        if (input.inset == 0) // Drop shadow: rendered outside element
        {
            alpha *= (1.0 - elem_inside);
        }
        else // Inset shadow: rendered inside element
        {
            alpha *= elem_inside;
        }
    }

    return input.color * alpha;
}
